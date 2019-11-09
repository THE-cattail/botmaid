// Package random includes some functions relative to random.
package random

import (
	"crypto/rand"
	"math/big"
	"reflect"
)

// BigInt returns a random bigInt in [x..y], with the help of crypto/rand.
func BigInt(x, y *big.Int) *big.Int {
	if x.Cmp(y) > 0 {
		return big.NewInt(int64(0))
	}

	len := &big.Int{}
	len.Sub(y, x)
	len.Add(len, big.NewInt(int64(1)))

	ret, err := rand.Int(rand.Reader, len)
	ret.Add(ret, x)

	if err != nil {
		return big.NewInt(int64(0))
	}
	return ret
}

// Int64 returns a random int64 in [x..y].
func Int64(x, y int64) int64 {
	return BigInt(big.NewInt(x), big.NewInt(y)).Int64()
}

// Int returns a random integer in [x..y].
func Int(x, y int) int {
	return int(Int64(int64(x), int64(y)))
}

// Slice returns a random element of a slice of empty interface variables.
// If s is not a slice, the function will return a nil pointer.
func Slice(s interface{}) interface{} {
	if reflect.TypeOf(s).Kind() == reflect.Slice {
		t := reflect.ValueOf(s)
		return t.Index(Int(0, t.Len()-1)).Interface()
	}

	return nil
}
