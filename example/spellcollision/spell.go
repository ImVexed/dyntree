package main

import (
	"fmt"
	"image"
	"image/color"
	"math/rand"
	"time"

	"github.com/ImVexed/dyntree"
)

// This file is an example using dyntree to simulate a 2D lingering AoE spell causing damage
// over multiple ticks to a group of enemies

type LingeringAoESpell struct {
	duration time.Duration
	dps      float64
	position dyntree.Vec3
	radius   float64
}

func (l *LingeringAoESpell) Position() dyntree.Vec3 {
	return l.position
}

func (l *LingeringAoESpell) Radius() float64 {
	return l.radius
}

func (l *LingeringAoESpell) HitTestEx(m *Mob) bool {
	// Maybe factor in dodge, block, accuracy, etc. here

	distSq := (l.position.X-m.position.X)*(l.position.X-m.position.X) +
		(l.position.Y-m.position.Y)*(l.position.Y-m.position.Y)

	radSq := (l.radius + m.size) * (l.radius + m.size)

	return distSq == radSq || distSq < radSq
}

func (l *LingeringAoESpell) HitTest(b dyntree.BoundingBox) bool {
	return b.Intersects(dyntree.BoxFromEntity(l))
}

func (l *LingeringAoESpell) DrawImage(i *image.RGBA) {
	x, y, dx, dy := int(l.radius)-1, 0, 1, 1
	err := dx - (int(l.radius) * 2)

	c := color.RGBA{255, 255, 0, 255}
	for x > y {
		i.Set(int(l.position.X)+x, int(l.position.Y)+y, c)
		i.Set(int(l.position.X)+y, int(l.position.Y)+x, c)
		i.Set(int(l.position.X)-y, int(l.position.Y)+x, c)
		i.Set(int(l.position.X)-x, int(l.position.Y)+y, c)
		i.Set(int(l.position.X)-x, int(l.position.Y)-y, c)
		i.Set(int(l.position.X)-y, int(l.position.Y)-x, c)
		i.Set(int(l.position.X)+y, int(l.position.Y)-x, c)
		i.Set(int(l.position.X)+x, int(l.position.Y)-y, c)

		if err <= 0 {
			y++
			err += dy
			dy += 2
		}
		if err > 0 {
			x--
			dx += 2
			err += dx - (int(l.radius) * 2)
		}
	}
}

type MobEntity interface {
	Self() *Mob
}

type Mob struct {
	idx      int
	health   float64
	position dyntree.Vec3
	size     float64
}

func (m *Mob) Position() dyntree.Vec3 {
	return m.position
}

func (m *Mob) Radius() float64 {
	return m.size
}

func (m *Mob) Self() *Mob {
	return m
}

func main() {
	rand.Seed(int64(time.Now().Nanosecond()))
	entityCount := 500_000
	fmt.Printf("Allocating %d entities, this may take a moment...\n", entityCount)

	mobs := make([]*Mob, entityCount)
	tree := dyntree.NewTree()

	// Insert mobs into the scene
	for n := 0; n < entityCount; n++ {
		m := &Mob{
			idx:    n,
			health: float64(rand.Intn(120)), // 100 damage is dealt over 2 seconds, so only ~20% should survive
			size:   1,
			position: dyntree.Vec3{
				X: float64(rand.Intn(10_000)),
				Y: float64(rand.Intn(10_000)),
			},
		}
		mobs[n] = m
		tree.Add(m)
	}

	tickRate := time.Second / 30
	ticker := time.NewTicker(tickRate)

	spell := &LingeringAoESpell{
		duration: 2 * time.Second,
		dps:      50,
		position: dyntree.Vec3{
			X: float64(rand.Intn(10_000)),
			Y: float64(rand.Intn(10_000)),
		},
		radius: 1000,
	}

	// Store when the spell was casted so we know when to stop
	casted := time.Now()
	ticks := 0
	deadMobs := 0
	fmt.Println("Starting simulation loop!")
	for {
		delta := time.Since(<-ticker.C)
		ticks++
		if delta.Milliseconds() > 0 {
			// The ability to maintain the tickrate is highly dependent on the underlying machine and how poorly optimized my code is
			fmt.Println("WARN: Tick rate slipped ", delta)
		}

		// Traverse the tree and collect the entities from bounding boxes we collided with
		hits := tree.ConcurrentTraverse(spell.HitTest)

		if time.Since(casted) > spell.duration {
			break
		}

		for _, e := range hits {
			m := e.(MobEntity).Self()

			// Do a higher precision hit test here now that we have a list of entities that we have likely colided with.
			// In our case it's an arguably simpler colision check than what's used in the tree, however, normally
			// you would do expensive things here that you couldn't afford to do on the whole tree of entities
			if !spell.HitTestEx(m) {
				continue
			}

			if m.health >= 0 {
				m.health -= (spell.dps / float64(time.Second.Milliseconds())) * float64((tickRate + delta).Milliseconds())
				if m.health < 0 {
					m.health = 0
					// Remove the mob from the collision tree once it has died
					tree.Remove(e)
					deadMobs++
				}
			}
		}
	}

	fmt.Printf("Spell ended, %d ticks in %s, %d out of %d mobs killed\n", ticks, time.Since(casted), deadMobs, entityCount)
	fmt.Println("Dumping image of tree at ./spell.bmp")
	// Add our spell so it will get it's DrawImage function called when we're saving an image of the tree
	tree.Add(spell)
	tree.Image("./spell.bmp")
}
