package random

// Word is a string with weight so that we can choose them randomly.
type Word struct {
	Word   string
	Weight int
}

// WordSlice is a slice of Word.
type WordSlice []Word

// Random returns a random string from the WordSlice.
func (ws WordSlice) Random() string {
	sum := 0
	for _, v := range ws {
		if v.Weight > 0 {
			sum += v.Weight
		}
	}
	if sum == 0 {
		return ""
	}

	r := Int(1, sum)
	t := 0
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
