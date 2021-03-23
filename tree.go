package dyntree

import (
	"image"
	"image/color"
	"image/draw"
	"math"
	"os"
	"sort"

	log "github.com/sirupsen/logrus"
	"golang.org/x/image/bmp"
)

//region
type Axis int
type Rot int
type NodeState int
type NodeSide int

const (
	X Axis = iota
	Y
	Z

	ROTNONE Rot = iota
	LEFTRIGHTLEFT
	LEFTRIGHTRIGHT
	RIGHTLEFTLEFT
	RIGHTLEFTRIGHT
	LEFTLEFTRIGHTRIGHT
	LEFTLEFTRIGHTLEFT

	OPTIMIZATIONQUEUED NodeState = iota

	LEFT NodeSide = iota
	RIGHT
)

func (a Axis) Next() Axis {
	switch a {
	case X:
		return Y
	case Y:
		return Z
	case Z:
		return X
	default:
		return X
	}
}

type RotOpt struct {
	Rot Rot
	SA  float64
}

func (best *RotOpt) FindBestRotation(n *Node, r Rot, sa float64) {
	new := GetRotationSurfaceArea(n, r, sa)

	if new.SA < best.SA {
		best.Rot = new.Rot
		best.SA = new.SA
	}
}

func GetRotationSurfaceArea(n *Node, rot Rot, sa float64) RotOpt {
	switch rot {
	case ROTNONE:
		return RotOpt{ROTNONE, sa}
	case LEFTRIGHTLEFT:
		if n.Right.IsLeaf() {
			return RotOpt{ROTNONE, math.MaxFloat64}
		} else {
			return RotOpt{rot, n.Right.Left.Box.SurfaceArea() + n.Left.Box.Expand(n.Right.Right.Box).SurfaceArea()}
		}
	case LEFTRIGHTRIGHT:
		if n.Right.IsLeaf() {
			return RotOpt{ROTNONE, math.MaxFloat64}
		} else {
			return RotOpt{rot, n.Right.Right.Box.SurfaceArea() + n.Left.Box.Expand(n.Right.Left.Box).SurfaceArea()}
		}
	case RIGHTLEFTLEFT:
		if n.Left.IsLeaf() {
			return RotOpt{ROTNONE, math.MaxFloat64}
		} else {
			return RotOpt{rot, n.Left.Left.Box.SurfaceArea() + n.Right.Box.Expand(n.Left.Right.Box).SurfaceArea()}
		}
	case RIGHTLEFTRIGHT:
		if n.Left.IsLeaf() {
			return RotOpt{ROTNONE, math.MaxFloat64}
		} else {
			return RotOpt{rot, n.Left.Right.Box.SurfaceArea() + n.Right.Box.Expand(n.Left.Left.Box).SurfaceArea()}
		}
	case LEFTLEFTRIGHTRIGHT:
		if n.Left.IsLeaf() || n.Right.IsLeaf() {
			return RotOpt{ROTNONE, math.MaxFloat64}
		} else {
			return RotOpt{rot, n.Right.Right.Box.Expand(n.Left.Right.Box).SurfaceArea() + n.Right.Left.Box.Expand(n.Left.Left.Box).SurfaceArea()}
		}
	case LEFTLEFTRIGHTLEFT:
		if n.Left.IsLeaf() || n.Right.IsLeaf() {
			return RotOpt{ROTNONE, math.MaxFloat64}
		} else {
			return RotOpt{rot, n.Right.Left.Box.Expand(n.Left.Right.Box).SurfaceArea() + n.Right.Right.Box.Expand(n.Left.Left.Box).SurfaceArea()}
		}
	default:
		panic("not implemented")
	}
}

type Vec3 struct {
	X, Y, Z float64
}

type BoundingBox struct {
	Min, Max Vec3
}

func (b BoundingBox) Intersects(b2 BoundingBox) bool {
	return ((b.Max.X > b2.Min.X) && (b.Min.X < b2.Max.X) &&
		(b.Max.Y > b2.Min.Y) && (b.Min.Y < b2.Max.Y) &&
		(b.Max.Z > b2.Min.Z) && (b.Min.Z < b2.Max.Z))
}

func (b BoundingBox) Equals(b2 BoundingBox) bool {
	return (b.Min.X == b2.Min.X) &&
		(b.Min.Y == b2.Min.Y) &&
		(b.Min.Z == b2.Min.Z) &&
		(b.Max.X == b2.Max.X) &&
		(b.Max.Y == b2.Max.Y) &&
		(b.Max.Z == b2.Max.Z)
}

func (b BoundingBox) SurfaceArea() float64 {
	xSize := b.Max.X - b.Min.X
	ySize := b.Max.Y - b.Min.Y
	zSize := b.Max.Z - b.Min.Z
	return 2.0 * (xSize*ySize + xSize*zSize + ySize*zSize)
}

func EntitiesSurfaceArea(ea []Entity, start, ct int) float64 {
	box := BoxFromEntity(ea[0])

	for i := start + 1; i < ct; i++ {
		box = box.Expand(BoxFromEntity(ea[i]))
	}

	return box.SurfaceArea()
}

func (b BoundingBox) Expand(b2 BoundingBox) BoundingBox {
	newbox := b

	if b2.Min.X < newbox.Min.X {
		newbox.Min.X = b2.Min.X
	}
	if b2.Min.Y < newbox.Min.Y {
		newbox.Min.Y = b2.Min.Y
	}
	if b2.Min.Z < newbox.Min.Z {
		newbox.Min.Z = b2.Min.Z
	}
	if b2.Max.X > newbox.Max.X {
		newbox.Max.X = b2.Max.X
	}
	if b2.Max.Y > newbox.Max.Y {
		newbox.Max.Y = b2.Max.Y
	}
	if b2.Max.Z > newbox.Max.Z {
		newbox.Max.Z = b2.Max.Z
	}

	return newbox
}

func BoxFromEntity(e Entity) BoundingBox {
	pos := e.Position()
	radius := e.Radius()

	box := BoundingBox{}
	box.Min.X = pos.X - radius
	box.Max.X = pos.X + radius
	box.Min.Y = pos.Y - radius
	box.Max.Y = pos.Y + radius
	box.Min.Z = pos.Z - radius
	box.Max.Z = pos.Z + radius
	return box
}

type SplitAxisOpt struct {
	Axis       Axis
	Items      []Entity
	SplitIndex int
	SA         float64
	HasValue   bool
}

func (s *SplitAxisOpt) LeftStartIndex() int {
	return 0
}
func (s *SplitAxisOpt) LeftEndIndex() int {
	return s.SplitIndex - 1
}
func (s *SplitAxisOpt) LeftItemCount() int {
	return s.LeftEndIndex()
}
func (s *SplitAxisOpt) RightStartIndex() int {
	return s.SplitIndex
}
func (s *SplitAxisOpt) RightEndIndex() int {
	return len(s.Items) - 1
}
func (s *SplitAxisOpt) RightItemCount() int {
	return s.RightEndIndex() - s.RightStartIndex()
}
func (s *SplitAxisOpt) TryImproveAxis(a Axis) {
	switch a {
	case X:
		sort.SliceStable(s.Items, func(i, j int) bool { return s.Items[i].Position().X < s.Items[j].Position().X })
	case Y:
		sort.SliceStable(s.Items, func(i, j int) bool { return s.Items[i].Position().Y < s.Items[j].Position().Y })
	case Z:
		sort.SliceStable(s.Items, func(i, j int) bool { return s.Items[i].Position().Z < s.Items[j].Position().Z })
	}

	left := EntitiesSurfaceArea(s.Items, s.LeftStartIndex(), s.LeftItemCount())
	right := EntitiesSurfaceArea(s.Items, s.RightStartIndex(), s.RightItemCount())
	new := left*float64(s.LeftItemCount()) + right*float64(s.RightItemCount())

	if !s.HasValue || new < s.SA {
		s.SA = new
		s.Axis = a
		s.HasValue = true
	}
}

type Entity interface {
	Position() Vec3
	Radius() float64
}

type Node struct {
	Box BoundingBox

	Parent *Node
	Left   *Node
	Right  *Node

	Depth       int
	NodeIndex   int
	BucketIndex int

	State NodeState
}

func (n *Node) IsLeaf() bool {
	return n.BucketIndex != -1
}

func (n *Node) HasParent() bool {
	return n.Parent != nil
}

func (n *Node) IsValid() bool {
	return n.IsValidLeafNode() || n.IsValidBranchNode()
}

func (n *Node) IsValidBranchNode() bool {
	return !n.IsLeaf() && n.Left != nil && n.Right != nil
}

func (n *Node) IsValidLeafNode() bool {
	return n.IsLeaf() && n.Left == nil && n.Right == nil
}

func (n *Node) IsValidBranch() bool {
	return n.IsValid() && (n.IsLeaf() || n.Right.IsValidBranch() && n.Left.IsValidBranch())
}

func (n *Node) Equals(n2 *Node) bool {
	return n.NodeIndex == n2.NodeIndex
}

func (n *Node) CompareDepth(n2 *Node) int {
	switch {
	case n.Depth < n2.Depth:
		return -1
	case n.Depth > n2.Depth:
		return 1
	default:
		return 0
	}
}

func (n *Node) AssignVolume(pos Vec3, radius float64) {
	n.Box.Min.X = pos.X - radius
	n.Box.Max.X = pos.X + radius
	n.Box.Min.Y = pos.Y - radius
	n.Box.Max.Y = pos.Y + radius
	n.Box.Min.Z = pos.Z - radius
	n.Box.Max.Z = pos.Z + radius
}

func (n *Node) ExpandVolume(pos Vec3, radius float64) {
	expanded := false

	if pos.X-radius < n.Box.Min.X {
		n.Box.Min.X = pos.X - radius
		expanded = true
	}

	if pos.X+radius > n.Box.Max.X {
		n.Box.Max.X = pos.X + radius
		expanded = true
	}

	if pos.Y-radius < n.Box.Min.Y {
		n.Box.Min.Y = pos.Y - radius
		expanded = true
	}

	if pos.Y+radius > n.Box.Max.Y {
		n.Box.Max.Y = pos.Y + radius
		expanded = true
	}

	if pos.Z-radius < n.Box.Min.Z {
		n.Box.Min.Z = pos.Z - radius
		expanded = true
	}

	if pos.Z+radius > n.Box.Max.Z {
		n.Box.Max.Z = pos.Z + radius
		expanded = true
	}

	if expanded && n.Parent != nil {
		n.Parent.ExpandParentVolume(n)
	}
}

func (n *Node) ExpandParentVolume(child *Node) {
	expanded := false
	if child.Box.Min.X < n.Box.Min.X {
		n.Box.Min.X = child.Box.Min.X
		expanded = true
	}

	if child.Box.Max.X > n.Box.Max.X {
		n.Box.Max.X = child.Box.Max.X
		expanded = true
	}

	if child.Box.Min.Y < n.Box.Min.Y {
		n.Box.Min.Y = child.Box.Min.Y
		expanded = true
	}

	if child.Box.Max.Y > n.Box.Max.Y {
		n.Box.Max.Y = child.Box.Max.Y
		expanded = true
	}

	if child.Box.Min.Z < n.Box.Min.Z {
		n.Box.Min.Z = child.Box.Min.Z
		expanded = true
	}

	if child.Box.Max.Z > n.Box.Max.Z {
		n.Box.Max.Z = child.Box.Max.Z
		expanded = true
	}

	if expanded && n.Parent != nil {
		n.Parent.ExpandParentVolume(n)
	}
}

func (n *Node) GetSibling() *Node {
	if n.Parent.Left.Equals(n) {
		return n.Parent.Right
	} else {
		return n.Parent.Left
	}
}

type HitTest func(box BoundingBox) bool

type Tree struct {
	rootNode *Node

	maxDepth  int
	maxLeaves int

	IsCreated bool

	leafs      map[Entity]*Node
	nodes      []*Node
	refitQueue []*Node

	unusedBucketIndicies []int
	unusedNodeIndicies   []int

	Buckets [][]Entity
}

func NewTree() *Tree {
	t := &Tree{
		maxLeaves: 1,

		leafs:      make(map[Entity]*Node),
		nodes:      make([]*Node, 0),
		refitQueue: make([]*Node, 0),

		unusedBucketIndicies: make([]int, 0),
		unusedNodeIndicies:   make([]int, 0),

		Buckets: make([][]Entity, 0),

		IsCreated: true,
	}

	t.rootNode = t.CreateNode(-1)

	return t
}

func (t *Tree) CreateNode(bucketIndex int) (n *Node) {
	index := 0
	if len(t.unusedNodeIndicies) > 0 {
		index, t.unusedNodeIndicies = t.unusedNodeIndicies[len(t.unusedNodeIndicies)-1], t.unusedNodeIndicies[:len(t.unusedNodeIndicies)-1]
		n = t.nodes[index]
	} else {
		n = &Node{}
		t.nodes = append(t.nodes, n)
		index = len(t.nodes)
	}

	n.NodeIndex = index
	n.BucketIndex = bucketIndex

	if n.BucketIndex == -1 {
		n.BucketIndex = t.GetOrCreateFreeBucket()
	}

	return
}

func (t *Tree) GetOrCreateFreeBucket() (index int) {
	if len(t.unusedBucketIndicies) > 0 {
		index, t.unusedBucketIndicies = t.unusedBucketIndicies[len(t.unusedBucketIndicies)-1], t.unusedBucketIndicies[:len(t.unusedBucketIndicies)-1]
		return
	}

	t.Buckets = append(t.Buckets, make([]Entity, 0))

	return len(t.Buckets)
}

func (t *Tree) FreeNode(n *Node) {
	n.Parent = nil
	n.Left = nil
	n.Right = nil
	n.BucketIndex = -1
	n.Depth = 0

	t.unusedNodeIndicies = append(t.unusedNodeIndicies, n.NodeIndex)
}

func (t *Tree) FreeBucket(n *Node) {
	t.unusedBucketIndicies = append(t.unusedBucketIndicies, n.BucketIndex)
	n.BucketIndex = -1
}

func (t *Tree) UnmapLeaf(e Entity) {
	delete(t.leafs, e)
}

func (t *Tree) MapLeaf(e Entity, n *Node) {
	t.leafs[e] = n
}

func (t *Tree) GetLeaf(e Entity) (n *Node, ok bool) {
	n, ok = t.leafs[e]
	return
}

func (t *Tree) QueueForOptimize(e Entity) bool {
	n, ok := t.GetLeaf(e)

	if !ok {
		return false
	}

	if !n.IsLeaf() {
		log.Errorln("Dangling leaf", n)
	}

	if bn, ok := t.TryFindBetterNode(n, e); ok {
		t.MoveItemBetweenNodes(n, bn, e)
	} else if t.RefitVolume(n) && n.Parent != nil {
		t.refitQueue = append(t.refitQueue, n)
	}

	return true
}

func (t *Tree) TraverseNode(cur *Node, test HitTest) (hits []Entity) {
	if !cur.IsValid() {
		return
	}

	if test(cur.Box) {
		if cur.BucketIndex != -1 {
			hits = append(hits, t.Buckets[cur.BucketIndex-1]...)
		}

		if cur.Left != nil {
			hits = append(hits, t.TraverseNode(cur.Left, test)...)
		}

		if cur.Right != nil {
			hits = append(hits, t.TraverseNode(cur.Right, test)...)
		}
	}

	return
}

func (t *Tree) Traverse(test HitTest) []Entity {
	return t.TraverseNode(t.rootNode, test)
}

func (t *Tree) TryFindBetterNode(cur *Node, e Entity) (bn *Node, ok bool) {
	box := BoxFromEntity(e)
	sa := box.SurfaceArea()

	bn = t.rootNode

	for bn.BucketIndex == -1 {
		if !bn.IsValid() || bn.Left != nil && !bn.Left.IsValid() || bn.Right != nil && !bn.Right.IsValid() {
			panic("Invalid node")
		}

		left := bn.Left
		right := bn.Right
		leftSa := right.Box.SurfaceArea() + left.Box.Expand(box).SurfaceArea()
		rightSa := left.Box.SurfaceArea() + right.Box.Expand(box).SurfaceArea()
		mergedSa := left.Box.Expand(right.Box).SurfaceArea() + sa

		// Doing a merge-and-pushdown can be expensive, so we only do it if it's notably better
		if mergedSa < math.Min(leftSa, rightSa)*0.3 {
			break
		}

		switch {
		case leftSa < rightSa:
			bn = left
		case leftSa > rightSa:
			bn = right
		}
	}

	if bn.Equals(t.rootNode) || bn.Equals(cur) {
		return nil, false
	}

	if bn.Equals(cur.Parent) && cur.IsLeaf() {
		// This scenario doesn't work because the source is a leaf and the parent already has two nodes,
		// so moving it up would create a dangling leaf in the vacated spot.
		// todo: might need to allow this where max leaf count > 1 and parent items bucket is not at max capacity.
		/*
		     x           x
		    / \         / \
		   ?   d       ?   s
		      / \	       /
		     ?   s       ?
		*/
		return nil, false
	}

	return bn, true
}

func (t *Tree) RemoveNode(n *Node) *Node {
	p := n.Parent
	gp := p.Parent

	keep := n.GetSibling()

	if gp == nil {
		if !keep.IsLeaf() {
			t.rootNode = keep
		}
	} else {
		keep.Parent = gp
		if gp.Left == p {
			gp.Left = keep
		} else {
			gp.Right = keep
		}
	}

	t.FreeBucket(n)
	t.FreeNode(n)
	t.FreeNode(p)

	if keep.BucketIndex != -1 {
		t.SetDepth(keep, keep.Depth+1)
	}

	if keep.Parent != nil {
		t.ChildRefit(keep.Parent, true)
	}

	return keep.Parent
}

func (t *Tree) MoveItemBetweenNodes(from, to *Node, e Entity) {
	t.RemoveItemFromNode(from, e)
	t.AddItemToNode(to, e)
}

func (t *Tree) RemoveItemFromNode(n *Node, e Entity) {
	if !n.IsLeaf() {
		panic("Remove on non leaf")
	}

	if !n.HasParent() {
		panic("attempt to collapse node with no parent")
	}

	t.UnmapLeaf(e)

	t.Buckets[n.BucketIndex-1] = append(t.Buckets[n.BucketIndex-1][:n.NodeIndex], t.Buckets[n.BucketIndex-1][n.NodeIndex+1:]...)

	if !t.IsEmpty(n) {
		t.RefitVolume(n)
	} else {
		if n.HasParent() {

		}
	}
}

func (t *Tree) IsEmpty(n *Node) bool {
	return !n.IsLeaf() || len(t.Buckets[n.BucketIndex-1]) == 0
}

func (t *Tree) AddItemToNode(n *Node, e Entity) {
	if n.IsLeaf() {
		t.AddItemToLeaf(n, e)
	} else {
		t.AddItemToBranch(n, e)
	}
}

func (t *Tree) AddItemToLeaf(n *Node, e Entity) {
	t.Buckets[n.BucketIndex-1] = append(t.Buckets[n.BucketIndex-1], e)
	t.MapLeaf(e, n)
	t.RefitVolume(n)
	t.SplitIfNecessary(n)
}

func (t *Tree) SplitIfNecessary(n *Node) {
	if t.ItemCount(n) > t.maxLeaves {
		t.SplitNode(n)
	}
}

func (t *Tree) ItemCount(n *Node) int {
	ct := 0
	if n.BucketIndex >= 0 {
		ct = len(t.Buckets[n.BucketIndex-1])
	}
	return ct
}

func (t *Tree) Optimize() {
	if t.maxLeaves != 1 {
		return
	}

	if len(t.refitQueue) == 0 {
		return
	}

	sort.SliceStable(t.refitQueue, func(i, j int) bool { return t.refitQueue[i].Depth < t.refitQueue[j].Depth })

	curDepth := t.refitQueue[0].Depth

	i := 0

	for curDepth > 0 {
		for ; i < len(t.refitQueue); i++ {
			n := t.refitQueue[i]

			if i == 0 {
				curDepth = n.Depth
			}

			if n.Depth != curDepth {
				break
			}

			if !n.IsValid() {
				continue
			}

			n.State &= OPTIMIZATIONQUEUED

			t.TryRotate(n)

			if !n.HasParent() {
				continue
			}

			if n.Parent.State&OPTIMIZATIONQUEUED != 0 {
				continue
			}

			n.Parent.State |= OPTIMIZATIONQUEUED

			t.refitQueue = append(t.refitQueue, n.Parent)
		}
		curDepth--
	}
	t.refitQueue = make([]*Node, 0)
}

func (t *Tree) TryRotate(n *Node) {
	if n.IsLeaf() && n.Parent != nil {
		return
	}

	sa := n.Left.Box.SurfaceArea() + n.Right.Box.SurfaceArea()
	best := &RotOpt{ROTNONE, math.MaxFloat64}

	best.FindBestRotation(n, LEFTRIGHTLEFT, sa)
	best.FindBestRotation(n, LEFTRIGHTRIGHT, sa)
	best.FindBestRotation(n, RIGHTLEFTLEFT, sa)
	best.FindBestRotation(n, RIGHTLEFTRIGHT, sa)
	best.FindBestRotation(n, LEFTLEFTRIGHTLEFT, sa)
	best.FindBestRotation(n, LEFTLEFTRIGHTRIGHT, sa)

	if best.Rot != ROTNONE {
		diff := (sa - best.SA) / sa
		if diff <= 0 {
			return
		}

		var swap *Node

		switch best.Rot {
		case ROTNONE:
			break
		case LEFTRIGHTLEFT:
			swap = n.Left
			n.Left = n.Right.Left
			n.Left.Parent = n
			n.Right.Left = swap
			swap.Parent = n.Right
			t.ChildRefit(n.Right, false)
			break

		case LEFTRIGHTRIGHT:
			swap = n.Left
			n.Left = n.Right.Right
			n.Left.Parent = n
			n.Right.Right = swap
			swap.Parent = n.Right
			t.ChildRefit(n.Right, false)
			break

		case RIGHTLEFTLEFT:
			swap = n.Right
			n.Right = n.Left.Left
			n.Right.Parent = n
			n.Left.Left = swap
			swap.Parent = n.Left
			t.ChildRefit(n.Left, false)
			break

		case RIGHTLEFTRIGHT:
			swap = n.Right
			n.Right = n.Left.Right
			n.Right.Parent = n
			n.Left.Right = swap
			swap.Parent = n.Left
			t.ChildRefit(n.Left, false)
			break

		case LEFTLEFTRIGHTRIGHT:
			swap = n.Left.Left
			n.Left.Left = n.Right.Right
			n.Right.Right = swap
			n.Left.Left.Parent = n.Left
			swap.Parent = n.Right
			t.ChildRefit(n.Left, false)
			t.ChildRefit(n.Right, false)
			break

		case LEFTLEFTRIGHTLEFT:
			swap = n.Left.Left
			n.Left.Left = n.Right.Left
			n.Right.Left = swap
			n.Left.Left.Parent = n.Left
			swap.Parent = n.Right
			t.ChildRefit(n.Left, false)
			t.ChildRefit(n.Right, false)
			break

		default:
			panic("not implemented")
		}

		switch best.Rot {
		case RIGHTLEFTRIGHT, RIGHTLEFTLEFT, LEFTRIGHTRIGHT, LEFTRIGHTLEFT:
			t.SetDepth(n, n.Depth)
			break
		}
	}

}

func (t *Tree) SplitNode(n *Node) {
	b := t.Buckets[n.BucketIndex-1]

	for _, e := range b {
		t.UnmapLeaf(e)
	}

	split := &SplitAxisOpt{
		Items:      b,
		SplitIndex: len(b) / 2,
	}

	split.TryImproveAxis(X)
	split.TryImproveAxis(Y)
	split.TryImproveAxis(Z)

	n.Left = t.CreateNodeFromSplit(n, split, LEFT, n.Depth+1, n.BucketIndex)
	n.Right = t.CreateNodeFromSplit(n, split, RIGHT, n.Depth+1, -1)
	n.BucketIndex = -1

	if !(!n.IsLeaf() && n.Left.IsLeaf() && n.Left.Left == nil && n.Left.Right == nil && n.Right.IsLeaf() && n.Right.Right == nil && n.Right.Left == nil) {
		panic("Invalid branch")
	}
}

func (t *Tree) CreateNodeFromSplit(parent *Node, split *SplitAxisOpt, side NodeSide, depth, bucketIndex int) *Node {
	new := t.CreateNode(bucketIndex)

	new.Parent = parent
	new.Depth = depth

	if t.maxDepth < depth {
		t.maxDepth = depth
	}

	if len(split.Items) < 1 {
		panic("No Items")
	}

	start := split.LeftStartIndex()
	end := split.LeftEndIndex()

	if side == RIGHT {
		start = split.RightStartIndex()
		end = split.RightEndIndex()
	}

	count := end - start + 1

	if side == LEFT && count <= len(t.Buckets[new.BucketIndex-1]) {
		t.Buckets[new.BucketIndex-1] = split.Items[start : end+1]
	} else {
		t.Buckets[new.BucketIndex-1] = append(t.Buckets[new.BucketIndex-1], split.Items[start:end+1]...)
	}

	for i := start; i <= end; i++ {
		t.MapLeaf(split.Items[i], new)
	}

	if count <= t.maxLeaves {
		new.Left = nil
		new.Right = nil
		t.ComputeVolume(new)
		t.SplitIfNecessary(new)
	} else {
		t.ComputeVolume(new)
		t.SplitIfNecessary(new)
		t.ChildRefit(new, false)
	}

	return new
}

func (t *Tree) RefitVolume(n *Node) bool {
	old := n.Box

	t.ComputeVolume(n)

	if !n.Box.Equals(old) {
		if n.Parent != nil {
			t.ChildRefit(n.Parent, true)
		}
		return true
	}
	return false
}

func (t *Tree) ComputeVolume(n *Node) {
	b := t.Buckets[n.BucketIndex-1]

	if len(b) == 0 {
		return
	}

	n.AssignVolume(b[0].Position(), b[0].Radius())

	for _, e := range b {
		n.ExpandVolume(e.Position(), e.Radius())
	}
}

func (t *Tree) ChildRefit(cur *Node, propogate bool) {
	for {
		cur.Box = cur.Left.Box.Expand(cur.Right.Box)

		cur = cur.Parent

		if !propogate || cur == nil {
			break
		}
	}
}

func (t *Tree) AddItemToBranch(n *Node, e Entity) {
	left := n.Left
	right := n.Right

	merged := t.CreateNode(-1)
	merged.Left = n.Left
	merged.Right = n.Right
	merged.Parent = n
	t.FreeBucket(merged)

	left.Parent = merged
	right.Parent = merged
	t.ChildRefit(merged, false)

	new := t.CreateNode(-1)
	new.Parent = n

	t.Buckets[new.BucketIndex-1] = append(t.Buckets[new.BucketIndex-1], e)

	t.MapLeaf(e, new)
	t.ComputeVolume(new)

	n.Left = merged
	n.Right = new

	t.SetDepth(n, n.Depth)
	t.ChildRefit(n, true)

}

func (t *Tree) SetDepth(n *Node, depth int) {
	n.Depth = depth
	if depth > t.maxDepth {
		t.maxDepth = depth
	}

	if !n.IsLeaf() {
		if !n.IsValidBranch() {
			panic("Bad branch")
		}
		t.SetDepth(n.Left, depth+1)
		t.SetDepth(n.Right, depth+1)
	}
}

func (t *Tree) AddObjectToNode(n *Node, e Entity, b BoundingBox, sa float64) {
	for n.BucketIndex == -1 {
		left := n.Left
		right := n.Right

		leftSa := left.Box.SurfaceArea()
		rightSa := right.Box.SurfaceArea()

		newLeftSA := rightSa + left.Box.Expand(b).SurfaceArea()
		newRightSA := leftSa + right.Box.Expand(b).SurfaceArea()
		merged := left.Box.Expand(right.Box).SurfaceArea() + sa

		if merged < math.Min(newLeftSA, newRightSA)*0.3 {
			t.AddItemToBranch(n, e)
			return
		}

		if newLeftSA < newRightSA {
			n = left
		} else {
			n = right
		}
	}

	t.Buckets[n.BucketIndex-1] = append(t.Buckets[n.BucketIndex-1], e)
	t.MapLeaf(e, n)
	t.RefitVolume(n)
	t.SplitIfNecessary(n)
}

func (t *Tree) Add(e Entity) {
	box := BoxFromEntity(e)
	t.AddObjectToNode(t.rootNode, e, box, box.SurfaceArea())
}

func (t *Tree) Image(path string) {
	frame := image.NewRGBA(image.Rect(int(t.rootNode.Box.Min.X), int(t.rootNode.Box.Min.Y), int(t.rootNode.Box.Max.X)+1, int(t.rootNode.Box.Max.Y)+1))
	draw.Draw(frame, frame.Bounds(), &image.Uniform{color.Black}, image.ZP, draw.Src)
	col := color.RGBA{255, 0, 0, 255}

	HLine := func(x1, y, x2 int) {
		for ; x1 <= x2; x1++ {
			frame.Set(x1, y, col)
		}
	}

	// VLine draws a veritcal line
	VLine := func(x, y1, y2 int) {
		for ; y1 <= y2; y1++ {
			frame.Set(x, y1, col)
		}
	}

	// Rect draws a rectangle utilizing HLine() and VLine()
	Rect := func(x1, y1, x2, y2 int) {
		HLine(x1, y1, x2)
		HLine(x1, y2, x2)
		VLine(x1, y1, y2)
		VLine(x2, y1, y2)
	}

	entities := t.Traverse(func(b BoundingBox) bool {
		Rect(int(b.Min.X), int(b.Min.Y), int(b.Max.X), int(b.Max.Y))
		return true
	})

	col = color.RGBA{0, 255, 0, 255}
	for _, e := range entities {
		b := BoxFromEntity(e)
		Rect(int(b.Min.X), int(b.Min.Y), int(b.Max.X), int(b.Max.Y))
	}

	f, _ := os.Create(path)
	bmp.Encode(f, frame)
}
