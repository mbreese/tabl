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
