package botmaid

import "github.com/catsworld/random"

// Word is a string with weight so that we can choose them randomly.
type Word struct {
	Word   string
	Weight int64
}

// RandomWordWithWeight returns a random string from the Word slice according
// to the weight.
func RandomWordWithWeight(ws []Word) string {
	sum := int64(0)
	for _, v := range ws {
		if v.Weight > 0 {
			sum += v.Weight
		}
	}
	if sum == 0 {
		return ""
	}

	r := random.Rand(1, sum)
	t := int64(0)
	for _, v := range ws {
		if v.Weight > 0 {
			t += v.Weight
			if t >= r {
				return v.Word
			}
		}
	}

	return ""
}
