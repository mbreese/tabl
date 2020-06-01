package support

// MaxInt - return the highest int
func MaxInt(nums ...int) int {
	v := nums[0]
	for i := 1; i < len(nums); i++ {
		if nums[i] > v {
			v = nums[i]
		}
	}
	return v
}

// MinInt - return the lowest int
func MinInt(nums ...int) int {
	v := nums[0]
	for i := 1; i < len(nums); i++ {
		if nums[i] < v {
			v = nums[i]
		}
	}
	return v
}

// BoolSum -- count the number of "true" values in a boolean array
func BoolSum(vals []bool) int {
	acc := 0
	for _, v := range vals {
		if v {
			acc++
		}
	}
	return acc
}
