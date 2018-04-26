package jsondiff

import (
	"encoding/json"
	"fmt"
	"log"
	"math/big"
	"strconv"
	"strings"
)

func logDebugf(fmt string, args ...interface{}) {
	log.Printf(fmt, args...)
}

func nopDebugf(fmt string, args ...interface{}) {}

var debugf = nopDebugf

// DiffType describes a difference type
type DiffType string

// Diff types
const (
	DiffIns  DiffType = "+"
	DiffDel  DiffType = "-"
	DiffMove DiffType = "<->"
	DiffMod  DiffType = "*"
)

// FieldName contains field name parts
type FieldName []string

func (f FieldName) String() string {
	return strings.Join(f, "/")
}

// Delta describes the difference between two corresponding nodes
type Delta interface {
	// GetType returns the type of delt
	GetType() DiffType
	// GetField returns the field name in the new copy, unless it is a
	// deletion, in which case the old field name is returned
	GetField() FieldName
}

// Insertion describes an insertion into an array, where NewNode is
// inserted into document 1 as Name
type Insertion struct {
	Name    FieldName
	NewNode interface{}
}

// GetField returns the inserted field name
func (x Insertion) GetField() FieldName { return x.Name }

// GetType returns the diff type
func (x Insertion) GetType() DiffType { return DiffIns }
func (x Insertion) String() string {
	return fmt.Sprintf("+ %s: %v", x.Name, x.NewNode)
}

// Deletion describes a deletion from an array, where DeletedNode is removed
// from document 1, and the removed field name name was Name
type Deletion struct {
	Name        FieldName
	DeletedNode interface{}
}

// GetField returns the deleted field name
func (x Deletion) GetField() FieldName { return x.Name }

// GetType returns the diff type
func (x Deletion) GetType() DiffType { return DiffDel }
func (x Deletion) String() string {
	return fmt.Sprintf("- %s: %v", x.Name, x.DeletedNode)
}

// Move describes an array element mode, where an element is moved from From to To
type Move struct {
	From FieldName
	To   FieldName
	Old  interface{}
	New  interface{}
}

// GetField returns the name of the destination field
func (x Move) GetField() FieldName { return x.To }

// GetType returns the diff type
func (x Move) GetType() DiffType { return DiffMove }
func (x Move) String() string {
	return fmt.Sprintf("<-> %s -> %s", x.From, x.To)
}

// Modification describes an edit where field is modified from Old to New
type Modification struct {
	Name FieldName
	Old  interface{}
	New  interface{}
}

// GetField returns the name of the modified field
func (x Modification) GetField() FieldName { return x.Name }

// GetType returns the diff type
func (x Modification) GetType() DiffType { return DiffMod }
func (x Modification) String() string {
	return fmt.Sprintf("* %s: (%v -> %v)", x.Name, x.Old, x.New)
}

//  Difference computes difference between two documents.
func JSONDifference(node1, node2 []byte) ([]Delta, error) {
	var n1, n2 interface{}
	err := json.Unmarshal(node1, &n1)
	if err != nil {
		return nil, nil
	}
	err = json.Unmarshal(node2, &n2)
	if err != nil {
		return nil, nil
	}
	return Difference(n1, n2), nil
}

// Difference computes difference between two documents. node1 and
// node2 are results of json.Unmarshal(&interface{})
func Difference(node1, node2 interface{}) []Delta {
	return nodeDifference(FieldName{}, node1, node2)
}

func nodeDifference(fieldName FieldName, node1, node2 interface{}) []Delta {
	if node1 == nil {
		if node2 == nil {
			return nil
		}
		return []Delta{Modification{Name: fieldName, Old: node1, New: node2}}
	}
	if node2 == nil {
		return []Delta{Modification{Name: fieldName, Old: node1, New: node2}}
	}
	// Both are non-nil
	switch n1 := node1.(type) {
	case map[string]interface{}:
		if n2, ok := node2.(map[string]interface{}); ok {
			return objectNodeDifference(fieldName, n1, n2)
		}
	case []interface{}:
		if n2, ok := node2.([]interface{}); ok {
			return arrayNodeDifference(fieldName, n1, n2)
		}
	default:
		return valueNodeDifference(fieldName, n1, node2)
	}
	return []Delta{Modification{Name: fieldName, Old: node1, New: node2}}
}

func objectNodeDifference(fieldName FieldName, node1, node2 map[string]interface{}) []Delta {
	var ret []Delta
	for key, v1 := range node1 {
		if v2, ok := node2[key]; ok {
			// Same field exists, compare
			d := nodeDifference(append(fieldName, key), v1, v2)
			if d != nil {
				ret = append(ret, d...)
			}
		} else {
			// Field does not exist on node2
			ret = append(ret, Modification{Name: append(fieldName, key),
				Old: v1,
				New: nil})
		}
	}
	for key, v2 := range node2 {
		_, ok := node1[key]
		if !ok {
			ret = append(ret, Modification{Name: append(fieldName, key),
				Old: nil,
				New: v2})
		}
	}
	return ret
}

func valueNodeDifference(fieldName FieldName, node1, node2 interface{}) []Delta {
	if node1 != node2 {
		return []Delta{Modification{Name: fieldName, Old: node1, New: node2}}
	}
	return nil
}

func arrayNodeDifference(fieldName FieldName, node1, node2 []interface{}) []Delta {
	return arrayDifference(fieldName, node1, node2, valueBasedEquivalence, false)
}

type dualMap struct {
	old2new map[int]int
	new2old map[int]int
}

func (x dualMap) insert(oldix, newix int) {
	x.old2new[oldix] = newix
	x.new2old[newix] = oldix
}

func (x dualMap) getNewIndex(oldix int) int {
	if i, ok := x.old2new[oldix]; ok {
		return i
	}
	return -1
}

func (x dualMap) getOldIndex(newix int) int {
	if i, ok := x.new2old[newix]; ok {
		return i
	}
	return -1
}

// valueBasedEquivalence compares nodes based on node values
func valueBasedEquivalence(node1, node2 []interface{}) dualMap {
	type nodeHashInfo struct {
		hash int
		eq   int
	}
	// Our goal is to compute an equivalence map.
	equivalence := dualMap{old2new: make(map[int]int), new2old: make(map[int]int)}
	// First step is to compute hashes on the nodes of node2.
	node2Hashes := make([]nodeHashInfo, len(node2))
	for i, n := range node2 {
		node2Hashes[i].hash = NodeHash(n)
		node2Hashes[i].eq = -1
	}
	// Then iterate node1 nodes, only comparing nodes from node2 whose
	// hashes match
	for i, n := range node1 {
		node1Hash := NodeHash(n)
		for j, h := range node2Hashes {
			if h.eq == -1 && node1Hash == h.hash {
				// these two nodes are possibly equal
				if IsEqual(n, node2[j]) {
					node2Hashes[j].eq = i
					equivalence.insert(i, j)
					break
				}
			}
		}
	}
	return equivalence
}

// arrayDifference computes difference between two array nodes based
// on array element values. Content equivalence cannot find
// differences inside an array node. It finds elements that are
// unmodified between the two arays, and assumes any other element is
// inserted/deleted. If the element indexes don't match, it assumes
// elements are moved
func arrayDifference(fieldName FieldName, node1, node2 []interface{},
	computeEq func(node1, node2 []interface{}) dualMap, recurse bool) []Delta {
	debugf("array diff n1: %v n2: %v", node1, node2)
	// Deal with trivial cases: if node1 is empty, then all node2 are additions
	// If node2 is empty, all node1 are deletions
	n1 := len(node1)
	n2 := len(node2)
	if n1 == 0 {
		ret := make([]Delta, n2)
		for i, x := range node2 {
			ret[i] = Insertion{Name: append(fieldName, strconv.Itoa(i)), NewNode: x}
		}
		return ret
	}
	if n2 == 0 {
		ret := make([]Delta, n1)
		for i, x := range node1 {
			ret[i] = Deletion{Name: append(fieldName, strconv.Itoa(i)), DeletedNode: x}
		}
		return ret
	}
	// Here, both arrays are nonempty

	equivalence := computeEq(node1, node2)

	debugf("Equivalences: %v", equivalence)
	ret := make([]Delta, 0)
	// If there is anything in node1 that's not contained in node2, thats a deletion
	for i := 0; i < n1; i++ {
		if equivalence.getNewIndex(i) == -1 {
			ret = append(ret, Deletion{Name: append(fieldName, strconv.Itoa(i)),
				DeletedNode: node1[i]})
		}
	}
	// If there is anything in node2 that's not in node1, that's an addition
	for i := 0; i < n2; i++ {
		if equivalence.getOldIndex(i) == -1 {
			ret = append(ret, Insertion{Name: append(fieldName, strconv.Itoa(i)),
				NewNode: node2[i]})
		}
	}

	pos1 := 0
	pos2 := 0
	// Keep recursively compared node2 indexes here to not duplicate comparisons
	recursedIndex := map[int]struct{}{}
	for {
		debugf("pos1: %d/%d pos2: %d/%d:", pos1, n1, pos2, n2)
		var oldix, newix int
		if pos1 < n1 {
			if pos2 < n2 {
				// Does the new node exist in the old node?
				oldix = equivalence.getOldIndex(pos2)
				debugf("pos2 %d -> oldix %d", pos2, oldix)
				if oldix == -1 {
					// This is a new item
					pos2++
				} else {
					if recurse {
						if _, ok := recursedIndex[pos2]; !ok {
							recursedIndex[pos2] = struct{}{}
							debugf("Recursively evaluating %d -> %d", pos2, oldix)
							rd := nodeDifference(append(fieldName, strconv.Itoa(pos2)), node1[oldix],
								node2[pos2])
							debugf("Result: %v", rd)
							if rd != nil {
								ret = append(ret, rd...)
							}
						}
					}
					// New node is in the old node. Make sure we take care of deletions
					newix = equivalence.getNewIndex(pos1)
					if newix == -1 {
						pos1++
					} else {
						// pos1: exists in node2 at index newix
						// pos2: exists in node1 at index oldix
						if oldix == pos1 {
							pos1++
							pos2++
						} else {
							ret = append(ret, Move{To: append(fieldName, strconv.Itoa(pos2)),
								From: append(fieldName, strconv.Itoa(oldix)),
								Old:  node1[oldix],
								New:  node2[pos2]})
							pos2++
						}
					}
				}
			} else {
				// These are all deleted items
				pos1++
			}
		} else if pos2 < n2 {
			// These are all insertions
			pos2++
		} else {
			break
		}
	}
	debugf("Result: %v", ret)
	return ret
}

// valueHash returns a hash for the given value. It is a weak has,
// but fast to compute. We are trying to find differences, not
// equivalences, so this is sufficient for our purposes
func valueHash(value interface{}) int {
	if value == nil {
		return 0
	}
	switch k := value.(type) {
	case bool:
		if k {
			return 1
		}
		return 0
	case int:
		return k
	case int8:
		return int(k)
	case int16:
		return int(k)
	case int32:
		return int(k)
	case int64:
		return int(k)
	case uint:
		return int(k)
	case uint8:
		return int(k)
	case uint16:
		return int(k)
	case uint32:
		return int(k)
	case uint64:
		return int(k)
	case float32:
		return int(k)
	case float64:
		return int(k)
	case big.Int:
		x := k.Int64()
		return int(x)
	case big.Float:
		x, _ := k.Int64()
		return int(x)
	case string:
		return stringHash(k)
	}
	return 0
}

// stringHash returns the sum of bytes in a string
func stringHash(s string) int {
	i := 0
	for _, c := range s {
		i += int(c)
	}
	return i
}

// objectNodeHash returns a hash value for an object node
func objectNodeHash(node map[string]interface{}) int {
	hash := 0
	for k, v := range node {
		hash += stringHash(k) + NodeHash(v)
	}
	return hash
}

// arrayNodeHash returns a hash value for an array node
func arrayNodeHash(node []interface{}) int {
	hash := 0
	for i, v := range node {
		hash += i * NodeHash(v)
	}
	return hash
}

// NodeHash calculates the hash of a node recursively
func NodeHash(node interface{}) int {
	if node == nil {
		return 0
	}
	switch k := node.(type) {
	case map[string]interface{}:
		return objectNodeHash(k)
	case []interface{}:
		return arrayNodeHash(k)
	}
	return valueHash(node)
}

// IsEqual checks if two nodes are the same
func IsEqual(node1, node2 interface{}) bool {
	if node1 == nil && node2 == nil {
		return true
	}
	if node1 == nil || node2 == nil {
		return false
	}
	switch k1 := node1.(type) {
	case map[string]interface{}:
		if x, ok := node2.(map[string]interface{}); ok {
			return isObjectNodeEqual(k1, x)
		}

	case []interface{}:
		if x, ok := node2.([]interface{}); ok {
			return isArrayNodeEqual(k1, x)
		}

	default:
		return k1 == node2
	}
	return false
}

func isObjectNodeEqual(node1, node2 map[string]interface{}) bool {
	if len(node1) != len(node2) {
		return false
	}
	for k, v := range node1 {
		if v2, ok := node2[k]; ok {
			if !IsEqual(v, v2) {
				return false
			}
		} else {
			return false
		}
	}
	return true
}

func isArrayNodeEqual(node1, node2 []interface{}) bool {
	if len(node1) != len(node2) {
		return false
	}
	for i, n1 := range node1 {
		if !IsEqual(n1, node2[i]) {
			return false
		}
	}
	return true
}
