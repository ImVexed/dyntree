package dyntree

import (
	"fmt"
	"math"
	"math/rand"
	"testing"
	"time"
)

type Person struct {
	size     float64
	position Vec3
}

func (p *Person) Position() Vec3 {
	return p.position
}

func (p *Person) Radius() float64 {
	return p.size
}

type Ray struct {
	Pos, Dir Vec3
}

func (r Ray) Intersects(b BoundingBox) bool {
	dirfrac := Vec3{}
	dirfrac.X = 1.0 / r.Dir.X
	dirfrac.Y = 1.0 / r.Dir.Y
	dirfrac.Z = 1.0 / r.Dir.Z
	// lb is the corner of AABB with minimal coordinates - left bottom, rt is maximal corner
	// r.org is origin of ray
	t1 := (b.Min.X - r.Pos.X) * dirfrac.X
	t2 := (b.Max.X - r.Pos.X) * dirfrac.X
	t3 := (b.Min.Y - r.Pos.Y) * dirfrac.Y
	t4 := (b.Max.Y - r.Pos.Y) * dirfrac.Y
	t5 := (b.Min.Z - r.Pos.Z) * dirfrac.Z
	t6 := (b.Max.Z - r.Pos.Z) * dirfrac.Z

	tmin := math.Max(math.Max(math.Min(t1, t2), math.Min(t3, t4)), math.Min(t5, t6))
	tmax := math.Min(math.Min(math.Max(t1, t2), math.Max(t3, t4)), math.Max(t5, t6))

	// if tmax < 0, ray (line) is intersecting AABB, but whole AABB is behing us
	if tmax < 0 {
		return false
	}

	// if tmin > tmax, ray doesn't intersect AABB
	if tmin > tmax {
		return false
	}

	return true
}

const AMMOUNT = 10000

func TestRay(T *testing.T) {
	t := NewTree()

	entities := make([]*Person, AMMOUNT)

	start := time.Now()
	for i := 0; i < AMMOUNT; i++ {
		entities[i] = &Person{
			size:     1,
			position: Vec3{float64(rand.Intn(1000)), float64(rand.Intn(1000)), 0},
		}
		t.Add(entities[i])
	}

	fmt.Println("Added in", time.Since(start))

	//t.Image("./map.bmp")

	gunshot := Ray{
		Pos: Vec3{0, 0, 0},
		Dir: Vec3{45, 45, 0},
	}

	start = time.Now()
	es := t.Traverse(gunshot.Intersects)
	fmt.Println("BVH: Nodes collided", len(es), "Elapsed", time.Since(start))
	bvhct := len(es)
	start = time.Now()
	es = []Entity{}

	for _, e := range entities {
		if gunshot.Intersects(BoxFromEntity(e)) {
			es = append(es, e)
		}
	}
	fmt.Println("Loop: Nodes collided", len(es), "Elapsed", time.Since(start))

	if bvhct != len(es) {
		T.Fatal("BVH/Loop disagree")
	}
}

func BenchmarkTree_Build(b *testing.B) {
	rand.Seed(1313131313)
	t := NewTree()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		p := &Person{
			size:     1,
			position: Vec3{float64(rand.Intn(10000)), float64(rand.Intn(10000)), float64(rand.Intn(10000))},
		}
		b.StartTimer()
		t.Add(p)
	}
}

func BenchmarkArray_Build(b *testing.B) {
	rand.Seed(1313131313)
	es := []Entity{}
	start := time.Now()

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		b.StopTimer()
		if time.Since(start) > 30*time.Second {
			b.SkipNow()
		}
		p := &Person{
			size:     1,
			position: Vec3{float64(rand.Intn(10000)), float64(rand.Intn(10000)), float64(rand.Intn(10000))},
		}
		b.StartTimer()
		es = append(es, p)
	}
}

func generateTree(count int) *Tree {
	rand.Seed(1313131313)
	t := NewTree()

	for n := 0; n < count; n++ {
		t.Add(&Person{
			size:     1,
			position: Vec3{float64(rand.Intn(10000)), float64(rand.Intn(10000)), float64(rand.Intn(10000))},
		})
	}

	return t
}

func bvhTraversal(b *testing.B, count int) {
	t := generateTree(count)

	gunshot := Ray{
		Pos: Vec3{0, 0, 0},
		Dir: Vec3{45, 45, 0},
	}

	// We don't want to benchmark creating random objects, only their registration into the tree
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		t.Traverse(gunshot.Intersects)
	}
}

func BenchmarkRayTraversalBVH_1000(b *testing.B)    { bvhTraversal(b, 1000) }
func BenchmarkRayTraversalBVH_10000(b *testing.B)   { bvhTraversal(b, 10000) }
func BenchmarkRayTraversalBVH_100000(b *testing.B)  { bvhTraversal(b, 100000) }
func BenchmarkRayTraversalBVH_1000000(b *testing.B) { bvhTraversal(b, 1000000) }

func loopTraversal(b *testing.B, count int) {
	t := generateTree(count)

	gunshot := Ray{
		Pos: Vec3{0, 0, 0},
		Dir: Vec3{45, 45, 0},
	}

	traverse := func(test HitTest) []Entity {
		es := []Entity{}
		for e := range t.leafs {
			if gunshot.Intersects(BoxFromEntity(e)) {
				es = append(es, e)
			}
		}
		return es
	}

	// We don't want to benchmark creating random objects, only their registration into the tree
	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		traverse(gunshot.Intersects)
	}
}

func BenchmarkRayTraversalLoop_1000(b *testing.B)    { loopTraversal(b, 1000) }
func BenchmarkRayTraversalLoop_10000(b *testing.B)   { loopTraversal(b, 10000) }
func BenchmarkRayTraversalLoop_100000(b *testing.B)  { loopTraversal(b, 100000) }
func BenchmarkRayTraversalLoop_1000000(b *testing.B) { loopTraversal(b, 1000000) }
