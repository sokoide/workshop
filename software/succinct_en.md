# Introduction to Succinct Data Structures (Go Perspective)

Succinct data structures are not just about data compression; they allow you to **search and operate on data directly in its compressed state (without deserialization)**. They represent data in a size close to its theoretical minimum (information-theoretic lower bound) while supporting operations like `rank/select/access` at high speed.

In this guide, we will first look at a standard `Trie`, then follow the flow of "converting a tree into a bit string" using `LOUDS`. The essence of Succinct is **replacing a forest of pointers with contiguous arrays and bit strings**.

---

## Prerequisites

### Succinct vs. Compressed

* **Succinct**: The original data is recoverable, and the size aims for `n + o(n)` bits for a theoretical lower bound `n`.
* **Compressed**: Becomes smaller on average, but worst-case scenario and operation costs are implementation-dependent.

### Why it Works in Practice

* Better memory locality, improving cache hit rates.
* Fewer pointer traversals, reducing CPU branch mispredictions.
* Effective for large-scale dictionaries, full-text search, and column-oriented data.

### 3 Key Points to Remember

* **Structure in bit strings**: Represent the shape of trees or arrays with bits instead of pointers.
* **Operations reduce to `rank/select`**: Determine "which child index" or "where the parent is" using bitwise operations.
* **Optimized for Reading**: Best for fixed dictionaries or read-heavy workloads.

---

## 1. Pointer-based Trie (Comparison Target)

This is a typical implementation. While easy to understand, it incurs significant overhead due to `map` and pointers for each node.

```go
package main

import "fmt"

type trieNode struct {
	children map[byte]*trieNode
	isWord   bool
}

type trie struct {
	root *trieNode
}

func newTrie() *trie {
	return &trie{root: &trieNode{children: make(map[byte]*trieNode)}}
}

func (t *trie) Insert(word string) {
	n := t.root
	for i := 0; i < len(word); i++ {
		c := word[i]
		if n.children[c] == nil {
			n.children[c] = &trieNode{children: make(map[byte]*trieNode)}
		}
		n = n.children[c]
	}
	n.isWord = true
}

func (t *trie) Search(word string) bool {
	n := t.root
	for i := 0; i < len(word); i++ {
		c := word[i]
		next := n.children[c]
		if next == nil {
			return false
		}
		n = next
	}
	return n.isWord
}

func main() {
	t := newTrie()
	for _, w := range []string{"app", "apple", "banana", "ball"} {
		t.Insert(w)
	}
	fmt.Println(t.Search("app"))   // true
	fmt.Println(t.Search("appl"))  // false
	fmt.Println(t.Search("apple")) // true
}
```

### Structure Image

```mermaid
graph TD
    R["root"]
    R --> A["a"]
    R --> B["b"]
    A --> AP1["p"]
    AP1 --> AP2["p"]
    AP2 --> ALEAF["* app"]
    AP2 --> L["l"]
    L --> E["e"]
    E --> APPLE["* apple"]
    B --> BA["a"]
    BA --> BAN["n..."]
    BA --> BAL["l..."]
```

While intuitive, the memory efficiency degrades as the number of words increases because each node holds a `map` and pointers.

### Why Pointer-based Implementation Consumes More Memory

The theoretical information of a `Trie` is simply "which node has which character branch" and "is it the end of a word." However, pointer-based implementations carry a lot of extra information for runtime convenience.

* Each node has pointers.
* The `map` itself has headers and buckets.
* Nodes are scattered on the heap, resulting in fine-grained allocations.
* The ratio of "management cost" to actual data tends to be large.

In short, you only want to represent the "shape of the tree," but you are paying a lot of memory for **pointers, hash tables, and allocator management information**.

---

## 2. Succinct Trie Concept (LOUDS)

LOUDS (Level-Order Unary Degree Sequence) arranges each node of the tree (Trie) in Breadth-First Search (BFS) order and represents the structure as a bit string instead of pointers.

* Represent the number of children for each node as `1...10` (a sequence of `1`s equal to the number of children, terminated by a `0`).
* Store edge labels (characters) in a separate array.
* Restore parent-child relationships using `rank/select`.

### Conversion Flow

```mermaid
flowchart LR
    T["Trie"] --> B["Arrange nodes in BFS order"]
    B --> L["Convert children count to LOUDS"]
    B --> C["Store edge labels in an array"]
    L --> Q["Restore parent-child with rank/select"]
    C --> Q
```

### Example (Concept)

```text
Children count per node: [3, 2, 1, 0, 1, 0, ...]
LOUDS:                   1110 110 10 0 10 0 ...
```

### Visualizing in ASCII

```text
Trie:
root
├─ a
│  └─ p
│     └─ p *
└─ b
   └─ a *

BFS Order:
[root][a][b][p][a][p]

Children Count:
root=2, a=1, b=1, p=1, a=0, p=0

LOUDS:
110 10 10 10 0 0
```

The reading method is simple: `110` for 2 children, `10` for 1 child, and `0` for 0 children. The shape of the tree is packed into this bit string, and character information is placed in a separate label array.

In production implementations, auxiliary indices (superblocks/blocks) are added to allow `rank/select` queries in approximately `O(1)` time. The essence is not linear search, but **fast positional queries against a bit string**.

### Benefits of LOUDS

LOUDS maps the tree structure into a contiguous bit string instead of pointers. This provides the following improvements:

* Eliminates the need for per-node pointers.
* No `map` is used, so hash table management costs disappear.
* Data is densely packed in arrays, leading to better cache locality.
* Tree structures can be handled with "arrays + bitwise operations," making it scalable for massive dictionaries.

In essence, while pointer-based trees are a "collection of objects," LOUDS treated the **tree as data that can be directly operated on as a serialized array, without needing to be deserialized**.

### Disadvantages of LOUDS

While memory efficient, implementation and updates are more difficult.

* Requires understanding and implementing `rank/select`, which has a high learning curve.
* Local updates (like adding a single node) are difficult; it usually requires reconstruction.
* Difficult to trace the structure visually during debugging.
* For small data, the effect might not justify the implementation complexity.
* If label search or auxiliary index design is incorrect, the expected speed will not be achieved.

---

## 3. Minimal Rank Implementation in Go (For Learning)

The following is a minimal example of "holding a bit string in `[]uint64`." `Rank1` uses linear scanning for educational purposes, but the implementation style is natural for Go.

```go
package succinct

import "math/bits"

type BitVector struct {
	words []uint64
	nbits int
}

func NewBitVector(nbits int) *BitVector {
	return &BitVector{
		words: make([]uint64, (nbits+63)/64),
		nbits: nbits,
	}
}

func (bv *BitVector) Set(i int) {
	if i < 0 || i >= bv.nbits {
		return
	}
	bv.words[i/64] |= 1 << (i % 64)
}

func (bv *BitVector) Get(i int) bool {
	if i < 0 || i >= bv.nbits {
		return false
	}
	return (bv.words[i/64]>>(i%64))&1 == 1
}

// Rank1 returns number of 1-bits in [0, i].
func (bv *BitVector) Rank1(i int) int {
	if bv.nbits == 0 {
		return 0
	}
	if i < 0 {
		return 0
	}
	if i >= bv.nbits {
		i = bv.nbits - 1
	}

	full := i / 64
	sum := 0
	for w := 0; w < full; w++ {
		sum += bits.OnesCount64(bv.words[w])
	}

	lastBits := (i % 64) + 1
	mask := ^uint64(0)
	if lastBits < 64 {
		mask = (uint64(1) << lastBits) - 1
	}
	sum += bits.OnesCount64(bv.words[full] & mask)
	return sum
}
```

### Key Understanding Points

* Pack bit strings into `[]uint64` instead of `[]bool`.
* Use `math/bits` to count the number of `1`s.
* In practice, instead of scanning everything in `Rank1`, auxiliary arrays are added for acceleration.

---

## 4. Complexity and Trade-offs

| Aspect | Pointer-based Trie | Succinct Trie |
| :--- | :--- | :--- |
| Space Efficiency | Low (Pointer/`map` overhead) | High (Aims for `n + o(n)`) |
| Ease of Implementation | High | Low |
| Updates (Insert/Delete) | Easy | Hard (Reconstruction cost) |
| Search Speed | Implementation Dependent | Often advantageous due to array cache locality |

### In a Nutshell

* **Pointer-based**: Easy to build and update, but consumes logic memory due to runtime overhead.
* **LOUDS**: Difficult to build, but extremely powerful for large-scale, read-mostly scenarios (saves memory, improves cache hits).

### Rough Estimate: 1 Million Words

Let's estimate with some assumptions:

* 1 million words.
* 1-16 characters per word.
* Average length is about 8.5 characters.
* Characters represented as `byte`.
* Assuming about 6 million Trie nodes after prefix sharing.
* Pointer-based uses `map[byte]*trieNode` version.

Total characters are approximately `1M x 8.5 = 8.5M`. With prefix sharing, we assume about 6 million nodes.

#### Pointer-based Trie Breakdown

* `trieNode` body: ~16 bytes per node.
* `map` header: ~48 bytes per node.
* Bucket area: Dozens to over 100 bytes per node.

Roughly, for 6 million nodes:

* Node bodies: `6M x 16B` = ~96 MB.
* `map` headers: `6M x 48B` = ~288 MB.
* Bucket area: ~400–900 MB.
* Runtime costs like GC/fragmentation: Dozens to hundreds of MB.

In total, it wouldn't be surprising if it reached **0.8 GB to 1.5 GB**.

#### LOUDS Trie Breakdown

LOUDS holds the same structure with "Bit string + Label array + Auxiliary index."

* LOUDS bit string: ~`2N` bits.
* Label array: ~1 byte per node.
* End-of-word flags: 1 bit per node.
* Auxiliary arrays for `rank/select`: A few % to dozens of %.

For `N = 6M`:

* LOUDS bit string: ~12M bits = ~1.5 MB.
* Label array: ~6M bytes = ~6 MB.
* End flags: ~6M bits = ~0.75 MB.
* Auxiliary index: ~1–5 MB.
* Implementation overhead: A few MB.

In total, it often fits within **10 MB to 20+ MB**.

#### Conclusion from Comparison

For the same 1 million words, a `map`-based pointer Trie can be **GB-class**, whereas LOUDS can be **dozens of MB**. In terms of memory efficiency, there can be a **50x to 100x+ difference** because LOUDS avoids the overhead of managing millions of small objects.

Of course, this is because the conditions are "read-mostly and almost immutable." If frequent updates are needed, choosing based on memory alone is dangerous.

### Visualizing Memory Difference

```mermaid
flowchart TD
    P["Pointer-based Trie\n~ 0.8 GB - 1.5 GB"] --> P1["Node bodies\n~ 96 MB"]
    P --> P2["map headers\n~ 288 MB"]
    P --> P3["Bucket area\n~ 400-900 MB"]
    P --> P4["GC / Fragmentation\nDecent amount"]

    L["LOUDS Trie\n~ 10 MB - 20+ MB"] --> L1["LOUDS bit string\n~ 1.5 MB"]
    L --> L2["Label array\n~ 6 MB"]
    L --> L3["End flags\n~ 0.75 MB"]
    L --> L4["Auxiliary index\n~ 1-5 MB"]
```

The essence of this difference is that while pointer-based implementations have per-node management structures (headers, pointers), LOUDS packs the entire tree into contiguous memory. When management costs dominate actual data, pointer-based structures become extremely inefficient.

**Criteria for Practical Decision**

* Fixed or near-fixed dictionaries/indices: Consider **Succinct**.
* Frequent updates or early stage of development: Prioritize **Standard Trie**.

### Usage Cheat Sheet

```mermaid
flowchart TD
    A["Is the data large?"] -->|Yes| B["Is it read-heavy?"]
    A -->|No| N["Standard structures are enough"]
    B -->|Yes| C["Is memory reduction critical?"]
    B -->|No| D["Standard Trie (better for updates)"]
    C -->|Yes| E["Consider Succinct"]
    C -->|No| F["Prioritize simple implementation"]
```

---

## 5. SWE Implementation Guide (Go)

* Start by creating a baseline with `[]uint64` + `bits.OnesCount64`.
* Measure memory and latency using `go test -bench . -benchmem`.
* Add index construction (auxiliary arrays for rank/select) in a separate phase.
* Separate APIs by purpose: `Builder` (construction) and `Reader` (lookup).
* If updates are necessary, consider `immutable snapshots` + differential reconstruction.

### Implementation Order

1. Create `BitVector`.
2. Implement `Rank/Select`.
3. Add tree structure using `LOUDS`.
4. Benchmark and compare with standard Trie.

---

## Summary

* Succinct is more about an **efficient, operable memory representation** than just "compression."
* In Go, favoring array-centric implementations yields better locality and predictability.
* Don't implement everything at once; introduce phases in the order of `BitVector -> rank/select -> LOUDS`.
