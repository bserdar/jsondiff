package jsondiff

import (
	"encoding/json"
	"testing"
)

func parse(s string) (interface{}, error) {
	var doc interface{}
	e := json.Unmarshal([]byte(s), &doc)
	return doc, e
}

func TestNoDiff(t *testing.T) {
	doc, err := parse(`{"f1":"value1","f2":2,"f3":null,"f4":true}`)
	if err != nil {
		t.Errorf("Cannot parse: %s", err)
		return
	}
	delta := Difference(doc, doc)
	if delta != nil {
		t.Errorf("Unexpected diff: %v", delta)
	}
}

func TestBasicDiff(t *testing.T) {
	doc1, err := parse(`{"f1":"value1","f2":2,"f3":null,"f4":true}`)
	if err != nil {
		t.Errorf("Cannot parse: %s", err)
		return
	}
	doc2, err := parse(`{"f1":"value2","f2":2,"f3":null,"f4":true}`)
	if err != nil {
		t.Errorf("Cannot parse: %s", err)
		return
	}
	delta := Difference(doc1, doc2)
	if len(delta) != 1 {
		t.Errorf("Unexpected diff: %v", delta)
	}
	if m, ok := delta[0].(Modification); ok {
		if m.Name.String() != "f1" ||
			m.Old.(string) != "value1" ||
			m.New.(string) != "value2" {
			t.Errorf("Wrong data: %v", m)
		}
	} else {
		t.Errorf("Wrong delta: %v", delta[0])
	}

	doc2, err = parse(`{"f1":"value1","f2":3,"f3":null,"f4":true}`)
	if err != nil {
		t.Errorf("Cannot parse: %s", err)
		return
	}
	delta = Difference(doc1, doc2)
	if len(delta) != 1 {
		t.Errorf("Unexpected diff: %v", delta)
	}
	if m, ok := delta[0].(Modification); ok {
		if m.Name.String() != "f2" ||
			m.Old.(float64) != 2 ||
			m.New.(float64) != 3 {
			t.Errorf("Wrong data: %v", m)
		}
	} else {
		t.Errorf("Wrong delta: %v", delta[0])
	}

	doc2, err = parse(`{"f1":"value1","f2":2,"f3":null,"f4":false}`)
	if err != nil {
		t.Errorf("Cannot parse: %s", err)
		return
	}
	delta = Difference(doc1, doc2)
	if len(delta) != 1 {
		t.Errorf("Unexpected diff: %v", delta)
	}
	if m, ok := delta[0].(Modification); ok {
		if m.Name.String() != "f4" ||
			m.Old.(bool) != true ||
			m.New.(bool) != false {
			t.Errorf("Wrong data: %v", m)
		}
	} else {
		t.Errorf("Wrong delta: %v", delta[0])
	}
}

func TestBasicArrayNoDiff(t *testing.T) {
	doc1, err := parse(`{"f1":[1,2,3,4,5,6]}`)
	if err != nil {
		t.Errorf("Cannot parse: %s", err)
		return
	}
	doc2, err := parse(`{"f1":[1,2,3,4,5,6]}`)
	if err != nil {
		t.Errorf("Cannot parse: %s", err)
		return
	}
	delta := Difference(doc1, doc2)
	if len(delta) != 0 {
		t.Errorf("Unexpected diff: %v", delta)
	}

}

func TestBasicArrayAdd(t *testing.T) {
	doc1, err := parse(`{"f1":[]}`)
	if err != nil {
		t.Errorf("Cannot parse: %s", err)
		return
	}
	doc2, err := parse(`{"f1":[1,2]}`)
	if err != nil {
		t.Errorf("Cannot parse: %s", err)
		return
	}
	delta := Difference(doc1, doc2)
	if len(delta) != 2 {
		t.Errorf("Unexpected diff: %v", delta)
	}

	if a, ok := delta[0].(Insertion); ok {
		if a.Name.String() != "f1/0" ||
			a.NewNode.(float64) != 1 {
			t.Errorf("Bad diff: %v", a)
		}
	} else {
		t.Errorf("Wrong delta: %v", delta)
	}
	if a, ok := delta[1].(Insertion); ok {
		if a.Name.String() != "f1/1" ||
			a.NewNode.(float64) != 2 {
			t.Errorf("Bad diff: %v", a)
		}
	} else {
		t.Errorf("Wrong delta: %v", delta)
	}
}

func TestBasicArrayDiff(t *testing.T) {

	doc1, err := parse(`{"f1":[1,2,3,4,5,6]}`)
	if err != nil {
		t.Errorf("Cannot parse: %s", err)
		return
	}
	doc2, err := parse(`{"f1":[1,3,8,4,6]}`)
	if err != nil {
		t.Errorf("Cannot parse: %s", err)
		return
	}
	delta := Difference(doc1, doc2)
	if len(delta) != 3 {
		t.Errorf("Unexpected diff: %v", delta)
	}
	if d, ok := delta[0].(Deletion); ok {
		if d.Name.String() != "f1/1" ||
			d.DeletedNode.(float64) != 2 {
			t.Errorf("Bad diff: %v", d)
		}
	} else {
		t.Errorf("Wrong delta: %v", delta)
	}
	if d, ok := delta[1].(Deletion); ok {
		if d.Name.String() != "f1/4" ||
			d.DeletedNode.(float64) != 5 {
			t.Errorf("Bad diff: %v", d)
		}
	} else {
		t.Errorf("Wrong delta: %v", delta)
	}
	if a, ok := delta[2].(Insertion); ok {
		if a.Name.String() != "f1/2" ||
			a.NewNode.(float64) != 8 {
			t.Errorf("Bad diff: %v", a)
		}
	} else {
		t.Errorf("Wrong delta: %v", delta)
	}
}

func TestObjArrayNoDiff(t *testing.T) {
	doc1, err := parse(`{"f1":[{"a":"b","c":1,"d":[1,2,3]},{"a":"e","c":2,"d":[4,5]}]}`)
	if err != nil {
		t.Errorf("Cannot parse: %s", err)
		return
	}
	doc2, err := parse(`{"f1":[{"a":"b","c":1,"d":[1,2,3]},{"a":"e","c":2,"d":[4,5]}]}`)
	if err != nil {
		t.Errorf("Cannot parse: %s", err)
		return
	}
	delta := Difference(doc1, doc2)
	if len(delta) != 0 {
		t.Errorf("Unexpected diff: %v", delta)
	}
}

func TestObjArrayIDNoDiff(t *testing.T) {
	doc1, err := parse(`{"f1":[{"_id":"1","a":"b","c":1,"d":[1,2,3]},{"_id":"2","a":"e","c":2,"d":[4,5]}]}`)
	if err != nil {
		t.Errorf("Cannot parse: %s", err)
		return
	}
	doc2, err := parse(`{"f1":[{"_id":"1","a":"b","c":1,"d":[1,2,3]},{"_id":"2","a":"e","c":2,"d":[4,5]}]}`)
	if err != nil {
		t.Errorf("Cannot parse: %s", err)
		return
	}
	delta := Difference(doc1, doc2)
	if len(delta) != 0 {
		t.Errorf("Unexpected diff: %v", delta)
	}
}

func TestObjArrayAdd(t *testing.T) {
	doc1, err := parse(`{"f1":[{"a":"b","c":1,"d":[1,2,3]},{"a":"e","c":2,"d":[4,5]}]}`)
	if err != nil {
		t.Errorf("Cannot parse: %s", err)
		return
	}
	doc2, err := parse(`{"f1":[{"a":"b","c":1,"d":[1,2,3]},{"a":"f","c":2,"d":[4,5]},{"a":"e","c":2,"d":[4,5]}]}`)
	if err != nil {
		t.Errorf("Cannot parse: %s", err)
		return
	}
	delta := Difference(doc1, doc2)
	if len(delta) != 1 {
		t.Errorf("Unexpected diff: %v", delta)
	}

	if a, ok := delta[0].(Insertion); ok {
		if a.Name.String() != "f1/1" {
			t.Errorf("Bad diff: %v", a)
		}
	} else {
		t.Errorf("Wrong delta: %v", delta)
	}
}

func TestObjArrayIDAdd(t *testing.T) {
	doc1, err := parse(`{"f1":[{"_id":"1","a":"b","c":1,"d":[1,2,3]},{"_id":"2","a":"e","c":2,"d":[4,5]}]}`)
	if err != nil {
		t.Errorf("Cannot parse: %s", err)
		return
	}
	doc2, err := parse(`{"f1":[{"_id":"1","a":"b","c":1,"d":[1,2,3]},{"a":"f","c":2,"d":[4,5]},{"_id":"2","a":"e","c":2,"d":[4,5]}]}`)
	if err != nil {
		t.Errorf("Cannot parse: %s", err)
		return
	}
	delta := Difference(doc1, doc2)
	if len(delta) != 1 {
		t.Errorf("Unexpected diff: %v", delta)
	}

	if a, ok := delta[0].(Insertion); ok {
		if a.Name.String() != "f1/1" {
			t.Errorf("Bad diff: %v", a)
		}
	} else {
		t.Errorf("Wrong delta: %v", delta)
	}
}

func TestObjArrayDiff1(t *testing.T) {
	doc1, err := parse(`{"f1":[{"a":"b","c":1,"d":[1,2,3]},{"a":"e","c":2,"d":[4,5]}]}`)
	if err != nil {
		t.Errorf("Cannot parse: %s", err)
		return
	}
	doc2, err := parse(`{"f1":[{"a":"1"},{"a":"2"},{"a":"e","c":2,"d":[4,5]},{"a":"4"}]}`)
	if err != nil {
		t.Errorf("Cannot parse: %s", err)
		return
	}
	// First deletions, then additions
	delta := Difference(doc1, doc2)
	if len(delta) != 4 {
		t.Errorf("Unexpected diff: %v", delta)
	}

	if a, ok := delta[0].(Deletion); ok {
		if a.Name.String() != "f1/0" {
			t.Errorf("Bad deletion: %v", a)
		}
	} else {
		t.Errorf("Delete expected: %v", delta[0])
	}

	if a, ok := delta[1].(Insertion); ok {
		if a.Name.String() != "f1/0" {
			t.Errorf("Bad diff: %v", a)
		}
	} else {
		t.Errorf("Insert expectex: %v", delta[1])
	}
	if a, ok := delta[2].(Insertion); ok {
		if a.Name.String() != "f1/1" {
			t.Errorf("Bad diff: %v", a)
		}
	} else {
		t.Errorf("Insert expectex: %v", delta[2])
	}
	if a, ok := delta[3].(Insertion); ok {
		if a.Name.String() != "f1/3" {
			t.Errorf("Bad diff: %v", a)
		}
	} else {
		t.Errorf("Insert expectex: %v", delta[3])
	}
}

func TestObjArrayIDDiff1(t *testing.T) {
	doc1, err := parse(`{"f1":[{"_id":"1","a":"b","c":1,"d":[1,2,3]},{"_id":"2","a":"e","c":2,"d":[4,5]}]}`)
	if err != nil {
		t.Errorf("Cannot parse: %s", err)
		return
	}
	doc2, err := parse(`{"f1":[{"a":"1"},{"a":"2"},{"_id":"2","a":"e","c":2,"d":[4,5]},{"a":"4"}]}`)
	if err != nil {
		t.Errorf("Cannot parse: %s", err)
		return
	}
	// First deletions, then additions
	delta := Difference(doc1, doc2)
	if len(delta) != 4 {
		t.Errorf("Unexpected diff: %v", delta)
	}

	if a, ok := delta[0].(Deletion); ok {
		if a.Name.String() != "f1/0" {
			t.Errorf("Bad deletion: %v", a)
		}
	} else {
		t.Errorf("Delete expected: %v", delta[0])
	}

	if a, ok := delta[1].(Insertion); ok {
		if a.Name.String() != "f1/0" {
			t.Errorf("Bad diff: %v", a)
		}
	} else {
		t.Errorf("Insert expected: %v", delta[1])
	}
	if a, ok := delta[2].(Insertion); ok {
		if a.Name.String() != "f1/1" {
			t.Errorf("Bad diff: %v", a)
		}
	} else {
		t.Errorf("Insert expectex: %v", delta[2])
	}
	if a, ok := delta[3].(Insertion); ok {
		if a.Name.String() != "f1/3" {
			t.Errorf("Bad diff: %v", a)
		}
	} else {
		t.Errorf("Insert expected: %v", delta[3])
	}
}
