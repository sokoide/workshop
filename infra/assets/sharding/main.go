package main

import (
	"fmt"
	"hash/fnv"
	"sort"
	"strconv"
)

func hash64(s string) uint64 {
	h := fnv.New64a()
	_, _ = h.Write([]byte(s))
	return h.Sum64()
}

// ModRouter is the simplest hash-based router: hash(key) % N.
type ModRouter struct {
	numShards int
}

func NewModRouter(numShards int) *ModRouter {
	if numShards <= 0 {
		panic("numShards must be > 0")
	}
	return &ModRouter{numShards: numShards}
}

func (r *ModRouter) ShardFor(key string) int {
	return int(hash64(key) % uint64(r.numShards))
}

type vnode struct {
	token uint64
	node  string
}

// Ring is a minimal consistent-hash ring with virtual nodes.
type Ring struct {
	points []vnode
}

func NewRing(nodeNames []string, vnodesPerNode int) *Ring {
	if len(nodeNames) == 0 {
		panic("nodeNames must not be empty")
	}
	if vnodesPerNode <= 0 {
		panic("vnodesPerNode must be > 0")
	}

	points := make([]vnode, 0, len(nodeNames)*vnodesPerNode)
	for _, name := range nodeNames {
		for i := 0; i < vnodesPerNode; i++ {
			key := name + "#" + strconv.Itoa(i)
			points = append(points, vnode{
				token: hash64(key),
				node:  name,
			})
		}
	}

	sort.Slice(points, func(i, j int) bool {
		return points[i].token < points[j].token
	})

	return &Ring{points: points}
}

func (r *Ring) Get(key string) string {
	h := hash64(key)
	idx := sort.Search(len(r.points), func(i int) bool {
		return r.points[i].token >= h
	})
	if idx == len(r.points) {
		idx = 0
	}
	return r.points[idx].node
}

// GetN returns up to rf distinct nodes by walking the ring clockwise.
func (r *Ring) GetN(key string, rf int) []string {
	if rf <= 0 {
		return nil
	}
	h := hash64(key)
	idx := sort.Search(len(r.points), func(i int) bool {
		return r.points[i].token >= h
	})
	if idx == len(r.points) {
		idx = 0
	}

	seen := make(map[string]struct{}, rf)
	result := make([]string, 0, rf)

	for i := 0; i < len(r.points) && len(result) < rf; i++ {
		p := r.points[(idx+i)%len(r.points)]
		if _, ok := seen[p.node]; ok {
			continue
		}
		seen[p.node] = struct{}{}
		result = append(result, p.node)
	}
	return result
}

// TenantRouter maps tenants to dedicated rings and falls back to a default ring.
type TenantRouter struct {
	fallback *Ring
	rings    map[string]*Ring
}

func NewTenantRouter(fallback *Ring) *TenantRouter {
	if fallback == nil {
		panic("fallback ring must not be nil")
	}
	return &TenantRouter{
		fallback: fallback,
		rings:    make(map[string]*Ring),
	}
}

func (tr *TenantRouter) SetTenantRing(tenant string, ring *Ring) {
	if tenant == "" {
		panic("tenant must not be empty")
	}
	if ring == nil {
		panic("ring must not be nil")
	}
	tr.rings[tenant] = ring
}

func (tr *TenantRouter) Route(tenant, key string) string {
	if ring, ok := tr.rings[tenant]; ok {
		return ring.Get(key)
	}
	return tr.fallback.Get(key)
}

// SparseTable supports static range minimum query (RMQ).
type SparseTable struct {
	values []int
	lg     []int
	st     [][]int // st[k][i] holds the index of min in range [i, i+2^k-1]
}

func NewSparseTable(values []int) *SparseTable {
	n := len(values)
	if n == 0 {
		panic("values must not be empty")
	}

	lg := make([]int, n+1)
	for i := 2; i <= n; i++ {
		lg[i] = lg[i/2] + 1
	}

	kMax := lg[n] + 1
	st := make([][]int, kMax)
	st[0] = make([]int, n)
	for i := 0; i < n; i++ {
		st[0][i] = i
	}

	for k := 1; k < kMax; k++ {
		span := 1 << k
		half := span >> 1
		st[k] = make([]int, n-span+1)
		for i := 0; i+span <= n; i++ {
			left := st[k-1][i]
			right := st[k-1][i+half]
			if values[left] <= values[right] {
				st[k][i] = left
			} else {
				st[k][i] = right
			}
		}
	}

	return &SparseTable{values: values, lg: lg, st: st}
}

// RangeMinIndex returns the index of the minimum value in [l, r].
func (s *SparseTable) RangeMinIndex(l, r int) int {
	if l < 0 || r >= len(s.values) || l > r {
		panic("invalid range")
	}
	k := s.lg[r-l+1]
	left := s.st[k][l]
	right := s.st[k][r-(1<<k)+1]
	if s.values[left] <= s.values[right] {
		return left
	}
	return right
}

type minPair struct {
	idx int
	val int
}

func better(a, b minPair) minPair {
	if a.val < b.val {
		return a
	}
	if b.val < a.val {
		return b
	}
	if a.idx <= b.idx {
		return a
	}
	return b
}

// SegmentTree supports dynamic range minimum query with point updates.
type SegmentTree struct {
	n    int
	size int
	tree []minPair
}

func NewSegmentTree(values []int) *SegmentTree {
	n := len(values)
	if n == 0 {
		panic("values must not be empty")
	}
	size := 1
	for size < n {
		size <<= 1
	}

	inf := int(^uint(0) >> 1)
	tree := make([]minPair, size*2)
	for i := range tree {
		tree[i] = minPair{idx: -1, val: inf}
	}

	for i, v := range values {
		tree[size+i] = minPair{idx: i, val: v}
	}
	for i := size - 1; i >= 1; i-- {
		tree[i] = better(tree[i*2], tree[i*2+1])
	}

	return &SegmentTree{n: n, size: size, tree: tree}
}

func (t *SegmentTree) Update(idx, val int) {
	if idx < 0 || idx >= t.n {
		panic("index out of range")
	}
	p := t.size + idx
	t.tree[p] = minPair{idx: idx, val: val}
	for p > 1 {
		p >>= 1
		t.tree[p] = better(t.tree[p*2], t.tree[p*2+1])
	}
}

// RangeMin returns (index, value) of minimum in [l, r].
func (t *SegmentTree) RangeMin(l, r int) (int, int) {
	if l < 0 || r >= t.n || l > r {
		panic("invalid range")
	}
	l += t.size
	r += t.size

	inf := int(^uint(0) >> 1)
	leftBest := minPair{idx: -1, val: inf}
	rightBest := minPair{idx: -1, val: inf}

	for l <= r {
		if l%2 == 1 {
			leftBest = better(leftBest, t.tree[l])
			l++
		}
		if r%2 == 0 {
			rightBest = better(t.tree[r], rightBest)
			r--
		}
		l >>= 1
		r >>= 1
	}

	ans := better(leftBest, rightBest)
	return ans.idx, ans.val
}

func main() {
	fmt.Println("== 1) Basic modulo hashing ==")
	mod := NewModRouter(8)
	fmt.Printf("key=user:42 -> shard=%d\n", mod.ShardFor("user:42"))

	fmt.Println("\n== 2) Consistent hash ring ==")
	ring := NewRing([]string{"ingester-a", "ingester-b", "ingester-c", "ingester-d"}, 64)
	seriesKey := "tenant-a|service=api|metric=http_requests_total"
	fmt.Printf("primary node for %q -> %s\n", seriesKey, ring.Get(seriesKey))
	fmt.Printf("replication set (rf=3) -> %v\n", ring.GetN(seriesKey, 3))

	fmt.Println("\n== 3) Tenant-aware routing ==")
	premiumRing := NewRing([]string{"premium-a", "premium-b"}, 64)
	router := NewTenantRouter(ring)
	router.SetTenantRing("premium-tenant", premiumRing)
	fmt.Printf("tenant=free-tenant -> %s\n", router.Route("free-tenant", "user:1"))
	fmt.Printf("tenant=premium-tenant -> %s\n", router.Route("premium-tenant", "user:1"))

	fmt.Println("\n== 4) Sparse Table (static snapshot RMQ) ==")
	// Example: periodic snapshot of shard latency (ms).
	latencySnapshot := []int{14, 19, 11, 17, 13, 9, 16, 21}
	sparse := NewSparseTable(latencySnapshot)
	best := sparse.RangeMinIndex(2, 6)
	fmt.Printf("best shard index in window [2,6] -> %d (latency=%dms)\n", best, latencySnapshot[best])

	fmt.Println("\n== 5) Segment Tree (dynamic load RMQ) ==")
	// Example: live load scores for shards; lower is better.
	load := []int{55, 21, 43, 18, 62, 27, 35, 40}
	seg := NewSegmentTree(load)
	idx, val := seg.RangeMin(0, len(load)-1)
	fmt.Printf("least loaded shard before update -> %d (load=%d)\n", idx, val)

	seg.Update(3, 70) // shard 3 becomes busy
	idx, val = seg.RangeMin(0, len(load)-1)
	fmt.Printf("least loaded shard after update -> %d (load=%d)\n", idx, val)
}
