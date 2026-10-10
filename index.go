package buntdb

import (
	"strconv"
	"strings"

	"github.com/tidwall/gjson"
	"github.com/tidwall/grect"
)

// BitValue defines supported types for bit and low-cardinality indexing.
type BitValue interface {
	~int | ~int8 | ~int16 | ~int32 | ~int64 |
		~uint | ~uint8 | ~uint16 | ~uint32 | ~uint64 |
		~string | ~bool
}

// IndexBool returns true if a is less than b when parsed as boolean values (false < true).
func IndexBool(a, b string) bool {
	ba := parseBool(a)
	bb := parseBool(b)
	return !ba && bb
}

// IndexBit returns true if a is less than b when parsed as bit values (0 < 1).
func IndexBit(a, b string) bool {
	ba := parseBit(a)
	bb := parseBit(b)
	return ba < bb
}

func parseBool(s string) bool {
	trimmed := strings.TrimSpace(s)
	switch strings.ToLower(trimmed) {
	case "1", "t", "true", "yes", "y", "on":
		return true
	case "0", "f", "false", "no", "n", "off", "":
		return false
	}
	if b, err := strconv.ParseBool(trimmed); err == nil {
		return b
	}
	if n, err := strconv.ParseInt(trimmed, 0, 64); err == nil {
		return n != 0
	}
	return false
}

func parseBit(s string) uint64 {
	trimmed := strings.TrimSpace(s)
	if n, err := strconv.ParseUint(trimmed, 0, 64); err == nil {
		return n
	}
	if parseBool(trimmed) {
		return 1
	}
	return 0
}

func extractJSONField(doc, path string) (gjson.Result, bool) {
	if doc == "" {
		return gjson.Result{}, false
	}
	res := gjson.Get(doc, path)
	if res.Exists() {
		return res, true
	}
	trimmed := strings.TrimSpace(doc)
	if len(trimmed) > 0 && trimmed[0] != '{' && trimmed[0] != '[' {
		parsed := gjson.Parse(trimmed)
		if parsed.Exists() {
			return parsed, true
		}
		return gjson.Result{Type: gjson.String, Str: trimmed}, true
	}
	return gjson.Result{}, false
}

func resultToBool(res gjson.Result) bool {
	switch res.Type {
	case gjson.True:
		return true
	case gjson.False:
		return false
	case gjson.Number:
		return res.Int() != 0
	case gjson.String:
		return parseBool(res.Str)
	default:
		return false
	}
}

func resultToInt(res gjson.Result) int64 {
	switch res.Type {
	case gjson.Number:
		return res.Int()
	case gjson.True:
		return 1
	case gjson.False:
		return 0
	case gjson.String:
		if n, err := strconv.ParseInt(res.Str, 0, 64); err == nil {
			return n
		}
		if parseBool(res.Str) {
			return 1
		}
		return 0
	default:
		return 0
	}
}

func resultToBitFlag(res gjson.Result, bit uint) uint64 {
	if bit >= 64 {
		return 0
	}
	var val uint64
	switch res.Type {
	case gjson.Number:
		val = uint64(res.Int())
	case gjson.True:
		val = 1
	case gjson.False:
		val = 0
	case gjson.String:
		if n, err := strconv.ParseUint(res.Str, 0, 64); err == nil {
			val = n
		} else if parseBool(res.Str) {
			val = 1
		}
	default:
		return 0
	}
	return (val >> bit) & 1
}

func resultToMask(res gjson.Result, mask uint64) uint64 {
	var val uint64
	switch res.Type {
	case gjson.Number:
		val = uint64(res.Int())
	case gjson.True:
		val = 1
	case gjson.False:
		val = 0
	case gjson.String:
		if n, err := strconv.ParseUint(res.Str, 0, 64); err == nil {
			val = n
		} else if parseBool(res.Str) {
			val = 1
		}
	default:
		return 0
	}
	return val & mask
}

// IndexJSONBit provides index ordering for JSON boolean, bit, or low-cardinality status values.
func IndexJSONBit(path string) func(a, b string) bool {
	return func(a, b string) bool {
		resA, okA := extractJSONField(a, path)
		resB, okB := extractJSONField(b, path)
		if !okA && !okB {
			return false
		}
		if !okA {
			return true
		}
		if !okB {
			return false
		}
		if (resA.Type == gjson.True || resA.Type == gjson.False) &&
			(resB.Type == gjson.True || resB.Type == gjson.False) {
			bA := resultToBool(resA)
			bB := resultToBool(resB)
			return !bA && bB
		}
		if resA.Type == gjson.Number && resB.Type == gjson.Number {
			return resA.Int() < resB.Int()
		}
		if (resA.Type == gjson.Number || resA.Type == gjson.True || resA.Type == gjson.False) &&
			(resB.Type == gjson.Number || resB.Type == gjson.True || resB.Type == gjson.False) {
			return resultToInt(resA) < resultToInt(resB)
		}
		if resA.Type == gjson.String && resB.Type == gjson.String {
			nA, errA := strconv.ParseInt(resA.Str, 0, 64)
			nB, errB := strconv.ParseInt(resB.Str, 0, 64)
			if errA == nil && errB == nil {
				return nA < nB
			}
			bA, errBoolA := strconv.ParseBool(resA.Str)
			bB, errBoolB := strconv.ParseBool(resB.Str)
			if errBoolA == nil && errBoolB == nil {
				return !bA && bB
			}
			return IndexString(resA.Str, resB.Str)
		}
		return resultToInt(resA) < resultToInt(resB)
	}
}

// IndexJSONBitBool provides index ordering for boolean or bit values at the JSON path.
func IndexJSONBitBool(path string) func(a, b string) bool {
	return func(a, b string) bool {
		resA, okA := extractJSONField(a, path)
		resB, okB := extractJSONField(b, path)
		if !okA && !okB {
			return false
		}
		if !okA {
			return true
		}
		if !okB {
			return false
		}
		bA := resultToBool(resA)
		bB := resultToBool(resB)
		return !bA && bB
	}
}

// IndexJSONBitInt provides index ordering for integer bit values or status codes at the JSON path.
func IndexJSONBitInt(path string) func(a, b string) bool {
	return func(a, b string) bool {
		resA, okA := extractJSONField(a, path)
		resB, okB := extractJSONField(b, path)
		if !okA && !okB {
			return false
		}
		if !okA {
			return true
		}
		if !okB {
			return false
		}
		return resultToInt(resA) < resultToInt(resB)
	}
}

// IndexJSONBitString provides index ordering for status strings at the JSON path.
func IndexJSONBitString(path string) func(a, b string) bool {
	return func(a, b string) bool {
		resA, okA := extractJSONField(a, path)
		resB, okB := extractJSONField(b, path)
		if !okA && !okB {
			return false
		}
		if !okA {
			return true
		}
		if !okB {
			return false
		}
		return IndexString(resA.String(), resB.String())
	}
}

// IndexJSONBitFlag provides index ordering for a single bit at bit offset in an integer JSON field.
func IndexJSONBitFlag(path string, bit uint) func(a, b string) bool {
	return func(a, b string) bool {
		resA, okA := extractJSONField(a, path)
		resB, okB := extractJSONField(b, path)
		if !okA && !okB {
			return false
		}
		if !okA {
			return true
		}
		if !okB {
			return false
		}
		fA := resultToBitFlag(resA, bit)
		fB := resultToBitFlag(resB, bit)
		return fA < fB
	}
}

// IndexJSONBitMask provides index ordering for masked integer values in a JSON field.
func IndexJSONBitMask(path string, mask uint64) func(a, b string) bool {
	return func(a, b string) bool {
		resA, okA := extractJSONField(a, path)
		resB, okB := extractJSONField(b, path)
		if !okA && !okB {
			return false
		}
		if !okA {
			return true
		}
		if !okB {
			return false
		}
		mA := resultToMask(resA, mask)
		mB := resultToMask(resB, mask)
		return mA < mB
	}
}

// IndexJsonBit aliases IndexJSONBit.
func IndexJsonBit(path string) func(a, b string) bool {
	return IndexJSONBit(path)
}

// IndexJsonBitBool aliases IndexJSONBitBool.
func IndexJsonBitBool(path string) func(a, b string) bool {
	return IndexJSONBitBool(path)
}

// IndexJsonBitInt aliases IndexJSONBitInt.
func IndexJsonBitInt(path string) func(a, b string) bool {
	return IndexJSONBitInt(path)
}

// IndexJsonBitString aliases IndexJSONBitString.
func IndexJsonBitString(path string) func(a, b string) bool {
	return IndexJSONBitString(path)
}

// IndexJsonBitFlag aliases IndexJSONBitFlag.
func IndexJsonBitFlag(path string, bit uint) func(a, b string) bool {
	return IndexJSONBitFlag(path, bit)
}

// IndexJsonBitMask aliases IndexJSONBitMask.
func IndexJsonBitMask(path string, mask uint64) func(a, b string) bool {
	return IndexJSONBitMask(path, mask)
}

// IndexJSONRect converts a JSON property into rectangle coordinates for spatial indexing.
func IndexJSONRect(path string) func(item string) (min, max []float64) {
	return func(item string) (min, max []float64) {
		res, ok := extractJSONField(item, path)
		if !ok || !res.Exists() {
			return IndexRect(item)
		}
		if res.IsArray() {
			arr := res.Array()
			if len(arr) == 0 {
				return []float64{0}, []float64{0}
			}
			if len(arr) == 2 {
				p := []float64{arr[0].Float(), arr[1].Float()}
				return p, p
			}
			if len(arr) == 4 {
				return []float64{arr[0].Float(), arr[1].Float()}, []float64{arr[2].Float(), arr[3].Float()}
			}
			half := len(arr) / 2
			minVals := make([]float64, half)
			maxVals := make([]float64, half)
			for i := 0; i < half; i++ {
				minVals[i] = arr[i].Float()
				maxVals[i] = arr[half+i].Float()
			}
			return minVals, maxVals
		}
		r := grect.Get(res.String())
		return r.Min, r.Max
	}
}

// IndexJsonRect aliases IndexJSONRect.
func IndexJsonRect(path string) func(item string) (min, max []float64) {
	return IndexJSONRect(path)
}
