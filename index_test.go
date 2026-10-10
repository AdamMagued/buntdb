package buntdb

import (
	"strings"
	"testing"
)

func TestIndexBoolAndBit(t *testing.T) {
	if !IndexBool("false", "true") {
		t.Fatal("expected false < true")
	}
	if IndexBool("true", "false") {
		t.Fatal("expected !(true < false)")
	}
	if IndexBool("false", "false") {
		t.Fatal("expected !(false < false)")
	}
	if IndexBool("true", "true") {
		t.Fatal("expected !(true < true)")
	}
	if !IndexBool("0", "1") {
		t.Fatal("expected 0 < 1")
	}
	if !IndexBool("no", "yes") {
		t.Fatal("expected no < yes")
	}

	if !IndexBit("0", "1") {
		t.Fatal("expected 0 < 1")
	}
	if IndexBit("1", "0") {
		t.Fatal("expected !(1 < 0)")
	}
	if IndexBit("1", "1") {
		t.Fatal("expected !(1 < 1)")
	}
	if IndexBit("0", "0") {
		t.Fatal("expected !(0 < 0)")
	}
}

func TestIndexJSONBit_Basic(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.CreateIndex("active_idx", "*", IndexJSONBit("active")); err != nil {
		t.Fatal(err)
	}

	if err := db.Update(func(tx *Tx) error {
		tx.Set("1", `{"name":"alice","active":true}`, nil)
		tx.Set("2", `{"name":"bob","active":false}`, nil)
		tx.Set("3", `{"name":"charlie","active":true}`, nil)
		tx.Set("4", `{"name":"dave","active":false}`, nil)
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	var ordered []string
	if err := db.View(func(tx *Tx) error {
		return tx.Ascend("active_idx", func(key, value string) bool {
			ordered = append(ordered, key)
			return true
		})
	}); err != nil {
		t.Fatal(err)
	}

	// false comes before true, tie broken by key
	expected := "2,4,1,3"
	if strings.Join(ordered, ",") != expected {
		t.Fatalf("expected %s, got %s", expected, strings.Join(ordered, ","))
	}

	// Query with direct scalar pivot "true"
	var activeKeys []string
	if err := db.View(func(tx *Tx) error {
		return tx.AscendEqual("active_idx", "true", func(key, value string) bool {
			activeKeys = append(activeKeys, key)
			return true
		})
	}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(activeKeys, ",") != "1,3" {
		t.Fatalf("expected 1,3, got %s", strings.Join(activeKeys, ","))
	}

	// Query with JSON pivot
	var inactiveKeys []string
	if err := db.View(func(tx *Tx) error {
		return tx.AscendEqual("active_idx", `{"active":false}`, func(key, value string) bool {
			inactiveKeys = append(inactiveKeys, key)
			return true
		})
	}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(inactiveKeys, ",") != "2,4" {
		t.Fatalf("expected 2,4, got %s", strings.Join(inactiveKeys, ","))
	}
}

func TestIndexJSONBitBool(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.CreateIndex("bool_idx", "*", IndexJSONBitBool("verified")); err != nil {
		t.Fatal(err)
	}

	if err := db.Update(func(tx *Tx) error {
		tx.Set("1", `{"verified":1}`, nil)
		tx.Set("2", `{"verified":0}`, nil)
		tx.Set("3", `{"verified":true}`, nil)
		tx.Set("4", `{"verified":"false"}`, nil)
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	var ordered []string
	if err := db.View(func(tx *Tx) error {
		return tx.Ascend("bool_idx", func(key, value string) bool {
			ordered = append(ordered, key)
			return true
		})
	}); err != nil {
		t.Fatal(err)
	}

	expected := "2,4,1,3"
	if strings.Join(ordered, ",") != expected {
		t.Fatalf("expected %s, got %s", expected, strings.Join(ordered, ","))
	}

	var trueKeys []string
	if err := db.View(func(tx *Tx) error {
		return tx.AscendEqual("bool_idx", "1", func(key, value string) bool {
			trueKeys = append(trueKeys, key)
			return true
		})
	}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(trueKeys, ",") != "1,3" {
		t.Fatalf("expected 1,3, got %s", strings.Join(trueKeys, ","))
	}
}

func TestIndexJSONBitInt(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.CreateIndex("status_idx", "*", IndexJSONBitInt("status")); err != nil {
		t.Fatal(err)
	}

	// 0: pending, 1: processing, 2: shipped, 3: delivered
	if err := db.Update(func(tx *Tx) error {
		tx.Set("order:1", `{"orderId":"1","status":2}`, nil)
		tx.Set("order:2", `{"orderId":"2","status":0}`, nil)
		tx.Set("order:3", `{"orderId":"3","status":3}`, nil)
		tx.Set("order:4", `{"orderId":"4","status":1}`, nil)
		tx.Set("order:5", `{"orderId":"5","status":1}`, nil)
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	var ordered []string
	if err := db.View(func(tx *Tx) error {
		return tx.Ascend("status_idx", func(key, value string) bool {
			ordered = append(ordered, key)
			return true
		})
	}); err != nil {
		t.Fatal(err)
	}

	expected := "order:2,order:4,order:5,order:1,order:3"
	if strings.Join(ordered, ",") != expected {
		t.Fatalf("expected %s, got %s", expected, strings.Join(ordered, ","))
	}

	// Test AscendEqual for status 1
	var status1 []string
	if err := db.View(func(tx *Tx) error {
		return tx.AscendEqual("status_idx", "1", func(key, value string) bool {
			status1 = append(status1, key)
			return true
		})
	}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(status1, ",") != "order:4,order:5" {
		t.Fatalf("expected order:4,order:5, got %s", strings.Join(status1, ","))
	}

	// Test AscendRange for [1, 3)
	var rangeKeys []string
	if err := db.View(func(tx *Tx) error {
		return tx.AscendRange("status_idx", "1", "3", func(key, value string) bool {
			rangeKeys = append(rangeKeys, key)
			return true
		})
	}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(rangeKeys, ",") != "order:4,order:5,order:1" {
		t.Fatalf("expected order:4,order:5,order:1, got %s", strings.Join(rangeKeys, ","))
	}
}

func TestIndexJSONBitString(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.CreateIndex("state_idx", "*", IndexJSONBitString("state")); err != nil {
		t.Fatal(err)
	}

	if err := db.Update(func(tx *Tx) error {
		tx.Set("1", `{"state":"processing"}`, nil)
		tx.Set("2", `{"state":"canceled"}`, nil)
		tx.Set("3", `{"state":"confirmed"}`, nil)
		tx.Set("4", `{"state":"processing"}`, nil)
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	var ordered []string
	if err := db.View(func(tx *Tx) error {
		return tx.Ascend("state_idx", func(key, value string) bool {
			ordered = append(ordered, key)
			return true
		})
	}); err != nil {
		t.Fatal(err)
	}

	// canceled, confirmed, processing, processing
	expected := "2,3,1,4"
	if strings.Join(ordered, ",") != expected {
		t.Fatalf("expected %s, got %s", expected, strings.Join(ordered, ","))
	}

	var processing []string
	if err := db.View(func(tx *Tx) error {
		return tx.AscendEqual("state_idx", "processing", func(key, value string) bool {
			processing = append(processing, key)
			return true
		})
	}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(processing, ",") != "1,4" {
		t.Fatalf("expected 1,4, got %s", strings.Join(processing, ","))
	}
}

func TestIndexJSONBitFlagAndMask(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	// 1: flags = 1 (001b)
	// 2: flags = 2 (010b)
	// 3: flags = 5 (101b)
	// 4: flags = 6 (110b)
	if err := db.CreateIndex("bit0_idx", "*", IndexJSONBitFlag("flags", 0)); err != nil {
		t.Fatal(err)
	}
	if err := db.CreateIndex("bit1_idx", "*", IndexJSONBitFlag("flags", 1)); err != nil {
		t.Fatal(err)
	}
	if err := db.CreateIndex("mask_idx", "*", IndexJSONBitMask("flags", 0x03)); err != nil {
		t.Fatal(err)
	}

	if err := db.Update(func(tx *Tx) error {
		tx.Set("1", `{"flags":1}`, nil)
		tx.Set("2", `{"flags":2}`, nil)
		tx.Set("3", `{"flags":5}`, nil)
		tx.Set("4", `{"flags":6}`, nil)
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	// bit 0: 2 (0), 4 (0), 1 (1), 3 (1)
	var bit0 []string
	if err := db.View(func(tx *Tx) error {
		return tx.Ascend("bit0_idx", func(key, value string) bool {
			bit0 = append(bit0, key)
			return true
		})
	}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(bit0, ",") != "2,4,1,3" {
		t.Fatalf("expected 2,4,1,3, got %s", strings.Join(bit0, ","))
	}

	// Query bit0 == 1
	var bit0Set []string
	if err := db.View(func(tx *Tx) error {
		return tx.AscendEqual("bit0_idx", "1", func(key, value string) bool {
			bit0Set = append(bit0Set, key)
			return true
		})
	}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(bit0Set, ",") != "1,3" {
		t.Fatalf("expected 1,3, got %s", strings.Join(bit0Set, ","))
	}

	// bit 1: 1 (0), 3 (0), 2 (1), 4 (1)
	var bit1 []string
	if err := db.View(func(tx *Tx) error {
		return tx.Ascend("bit1_idx", func(key, value string) bool {
			bit1 = append(bit1, key)
			return true
		})
	}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(bit1, ",") != "1,3,2,4" {
		t.Fatalf("expected 1,3,2,4, got %s", strings.Join(bit1, ","))
	}

	// Mask 0x03:
	// 1: 1 & 3 = 1
	// 2: 2 & 3 = 2
	// 3: 5 & 3 = 1
	// 4: 6 & 3 = 2
	// Sorted: 1 and 3 (val 1), then 2 and 4 (val 2)
	var maskKeys []string
	if err := db.View(func(tx *Tx) error {
		return tx.Ascend("mask_idx", func(key, value string) bool {
			maskKeys = append(maskKeys, key)
			return true
		})
	}); err != nil {
		t.Fatal(err)
	}
	if strings.Join(maskKeys, ",") != "1,3,2,4" {
		t.Fatalf("expected 1,3,2,4, got %s", strings.Join(maskKeys, ","))
	}
}

func TestCompoundBitIndex(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.CreateIndex("active_age", "*", IndexJSONBitBool("active"), IndexJSON("age")); err != nil {
		t.Fatal(err)
	}

	if err := db.Update(func(tx *Tx) error {
		tx.Set("1", `{"active":true,"age":40}`, nil)
		tx.Set("2", `{"active":false,"age":30}`, nil)
		tx.Set("3", `{"active":true,"age":20}`, nil)
		tx.Set("4", `{"active":false,"age":50}`, nil)
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	var ordered []string
	if err := db.View(func(tx *Tx) error {
		return tx.Ascend("active_age", func(key, value string) bool {
			ordered = append(ordered, key)
			return true
		})
	}); err != nil {
		t.Fatal(err)
	}

	// false (age 30, then 50) -> 2, 4
	// true  (age 20, then 40) -> 3, 1
	expected := "2,4,3,1"
	if strings.Join(ordered, ",") != expected {
		t.Fatalf("expected %s, got %s", expected, strings.Join(ordered, ","))
	}

	// Descending age with active ascending
	if err := db.ReplaceIndex("active_age_desc", "*", IndexJSONBitBool("active"), Desc(IndexJSON("age"))); err != nil {
		t.Fatal(err)
	}
	var orderedDesc []string
	if err := db.View(func(tx *Tx) error {
		return tx.Ascend("active_age_desc", func(key, value string) bool {
			orderedDesc = append(orderedDesc, key)
			return true
		})
	}); err != nil {
		t.Fatal(err)
	}
	// false (age 50, then 30) -> 4, 2
	// true  (age 40, then 20) -> 1, 3
	expectedDesc := "4,2,1,3"
	if strings.Join(orderedDesc, ",") != expectedDesc {
		t.Fatalf("expected %s, got %s", expectedDesc, strings.Join(orderedDesc, ","))
	}
}

func TestIndexJSONRect(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.CreateSpatialIndex("loc_idx", "*", IndexJSONRect("location")); err != nil {
		t.Fatal(err)
	}

	if err := db.Update(func(tx *Tx) error {
		tx.Set("phx", `{"name":"Phoenix","location":[-112.074037,33.448377]}`, nil)
		tx.Set("sf", `{"name":"San Francisco","location":[-122.419416,37.774929]}`, nil)
		tx.Set("la", `{"name":"Los Angeles","location":[-118.243685,34.052234]}`, nil)
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	// Intersects query using BuntDB's Rect
	var found []string
	if err := db.View(func(tx *Tx) error {
		// Box covering Southern California and Arizona
		return tx.Intersects("loc_idx", Rect([]float64{-120, 32}, []float64{-110, 36}), func(key, value string) bool {
			found = append(found, key)
			return true
		})
	}); err != nil {
		t.Fatal(err)
	}

	if len(found) != 2 {
		t.Fatalf("expected 2 items, got %d", len(found))
	}
	if !((found[0] == "la" && found[1] == "phx") || (found[0] == "phx" && found[1] == "la")) {
		t.Fatalf("unexpected results: %v", found)
	}

	// Test 4-element box in JSON
	if err := db.CreateSpatialIndex("box_idx", "*", IndexJSONRect("bounds")); err != nil {
		t.Fatal(err)
	}
	if err := db.Update(func(tx *Tx) error {
		tx.Set("b1", `{"bounds":[-10, -10, 10, 10]}`, nil)
		tx.Set("b2", `{"bounds":[20, 20, 30, 30]}`, nil)
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	var boxFound []string
	if err := db.View(func(tx *Tx) error {
		return tx.Intersects("box_idx", Rect([]float64{0, 0}, []float64{5, 5}), func(key, value string) bool {
			boxFound = append(boxFound, key)
			return true
		})
	}); err != nil {
		t.Fatal(err)
	}
	if len(boxFound) != 1 || boxFound[0] != "b1" {
		t.Fatalf("expected [b1], got %v", boxFound)
	}
}

func TestIndexJSONBit_EdgeCases(t *testing.T) {
	db, err := Open(":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.CreateIndex("edge_idx", "*", IndexJsonBit("val")); err != nil {
		t.Fatal(err)
	}

	// Missing field vs present field
	if err := db.Update(func(tx *Tx) error {
		tx.Set("missing", `{"other":"data"}`, nil)
		tx.Set("zero", `{"val":0}`, nil)
		tx.Set("one", `{"val":1}`, nil)
		return nil
	}); err != nil {
		t.Fatal(err)
	}

	var keys []string
	if err := db.View(func(tx *Tx) error {
		return tx.Ascend("edge_idx", func(key, value string) bool {
			keys = append(keys, key)
			return true
		})
	}); err != nil {
		t.Fatal(err)
	}

	// missing is less than present
	if strings.Join(keys, ",") != "missing,zero,one" {
		t.Fatalf("expected missing,zero,one, got %s", strings.Join(keys, ","))
	}

	// Test bit index alias functions
	_ = IndexJsonBitBool("path")
	_ = IndexJsonBitInt("path")
	_ = IndexJsonBitString("path")
	_ = IndexJsonBitFlag("path", 0)
	_ = IndexJsonBitMask("path", 1)
	_ = IndexJsonRect("path")

	// Out-of-bounds bit offset
	f := IndexJSONBitFlag("val", 65)
	if f(`{"val":1}`, `{"val":1}`) {
		t.Fatal("expected false for out of bounds bit")
	}

	// Malformed JSON strings handled gracefully
	if f("not json", "also not json") {
		t.Fatal("expected false for non-json")
	}
}
