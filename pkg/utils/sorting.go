package utils

// SortStrings sorts a slice of strings in-place using quicksort
func SortStrings(arr []string) {
	if len(arr) < 2 {
		return
	}
	quickSortStrings(arr, 0, len(arr)-1)
}

func quickSortStrings(arr []string, low, high int) {
	if low < high {
		pivot := partitionStrings(arr, low, high)
		quickSortStrings(arr, low, pivot-1)
		quickSortStrings(arr, pivot+1, high)
	}
}

func partitionStrings(arr []string, low, high int) int {
	pivot := arr[high]
	i := low - 1

	for j := low; j < high; j++ {
		if arr[j] < pivot {
			i++
			arr[i], arr[j] = arr[j], arr[i]
		}
	}

	arr[i+1], arr[high] = arr[high], arr[i+1]
	return i + 1
}

// SortInt64Slice sorts a slice with associated data by int64 values
type Int64Sortable struct {
	Key   int64
	Value interface{}
}

// SortByInt64 sorts a slice of Int64Sortable in-place using quicksort
func SortByInt64(arr []Int64Sortable, ascending bool) {
	if len(arr) < 2 {
		return
	}
	quickSortInt64(arr, 0, len(arr)-1, ascending)
}

func quickSortInt64(arr []Int64Sortable, low, high int, ascending bool) {
	if low < high {
		pivot := partitionInt64(arr, low, high, ascending)
		quickSortInt64(arr, low, pivot-1, ascending)
		quickSortInt64(arr, pivot+1, high, ascending)
	}
}

func partitionInt64(arr []Int64Sortable, low, high int, ascending bool) int {
	pivot := arr[high]
	i := low - 1

	for j := low; j < high; j++ {
		shouldSwap := false
		if ascending {
			shouldSwap = arr[j].Key < pivot.Key
		} else {
			shouldSwap = arr[j].Key > pivot.Key
		}

		if shouldSwap {
			i++
			arr[i], arr[j] = arr[j], arr[i]
		}
	}

	arr[i+1], arr[high] = arr[high], arr[i+1]
	return i + 1
}

// SortFloat64Slice sorts a slice with associated data by float64 values
type Float64Sortable struct {
	Key   float64
	Value interface{}
}

// SortByFloat64 sorts a slice of Float64Sortable in-place using quicksort
func SortByFloat64(arr []Float64Sortable, ascending bool) {
	if len(arr) < 2 {
		return
	}
	quickSortFloat64(arr, 0, len(arr)-1, ascending)
}

func quickSortFloat64(arr []Float64Sortable, low, high int, ascending bool) {
	if low < high {
		pivot := partitionFloat64(arr, low, high, ascending)
		quickSortFloat64(arr, low, pivot-1, ascending)
		quickSortFloat64(arr, pivot+1, high, ascending)
	}
}

func partitionFloat64(arr []Float64Sortable, low, high int, ascending bool) int {
	pivot := arr[high]
	i := low - 1

	for j := low; j < high; j++ {
		shouldSwap := false
		if ascending {
			shouldSwap = arr[j].Key < pivot.Key
		} else {
			shouldSwap = arr[j].Key > pivot.Key
		}

		if shouldSwap {
			i++
			arr[i], arr[j] = arr[j], arr[i]
		}
	}

	arr[i+1], arr[high] = arr[high], arr[i+1]
	return i + 1
}

// HeapSort provides an alternative O(n log n) sorting algorithm
// Useful for large datasets where quicksort's worst case O(n²) is a concern

// SortStringsHeap sorts strings using heap sort
func SortStringsHeap(arr []string) {
	n := len(arr)

	// Build max heap
	for i := n/2 - 1; i >= 0; i-- {
		heapifyStrings(arr, n, i)
	}

	// Extract elements from heap one by one
	for i := n - 1; i > 0; i-- {
		arr[0], arr[i] = arr[i], arr[0]
		heapifyStrings(arr, i, 0)
	}
}

func heapifyStrings(arr []string, n, i int) {
	largest := i
	left := 2*i + 1
	right := 2*i + 2

	if left < n && arr[left] > arr[largest] {
		largest = left
	}

	if right < n && arr[right] > arr[largest] {
		largest = right
	}

	if largest != i {
		arr[i], arr[largest] = arr[largest], arr[i]
		heapifyStrings(arr, n, largest)
	}
}
