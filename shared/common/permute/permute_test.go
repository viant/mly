package permute

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test(t *testing.T) {
	p := NewPermuter([][]int{[]int{1, 2}, []int{3, 4}}, nil, [][]uint32{[]uint32{5, 6}})

	var c bool
	var k Permutation
	c, k = p.Next()
	require.True(t, c, k)
	require.Equal(t, []int{1, 3}, k.Ints)
	require.Equal(t, []uint32{5}, k.Uint32s)

	c, k = p.Next()
	require.True(t, c, k)
	require.Equal(t, []int{2, 3}, k.Ints)
	require.Equal(t, []uint32{5}, k.Uint32s)

	c, k = p.Next()
	require.True(t, c, k)
	require.Equal(t, []int{1, 4}, k.Ints)
	require.Equal(t, []uint32{5}, k.Uint32s)

	c, k = p.Next()
	require.True(t, c, k)
	require.Equal(t, []int{2, 4}, k.Ints)
	require.Equal(t, []uint32{5}, k.Uint32s)

	c, k = p.Next()
	require.True(t, c, k)
	require.Equal(t, []int{1, 3}, k.Ints)
	require.Equal(t, []uint32{6}, k.Uint32s)

	c, k = p.Next()
	require.True(t, c, k)

	c, k = p.Next()
	require.True(t, c, k)
	require.Equal(t, []int{1, 4}, k.Ints)
	require.Equal(t, []uint32{6}, k.Uint32s)

	c, k = p.Next()
	require.True(t, c, k)
	require.Equal(t, []int{2, 4}, k.Ints)

	c, k = p.Next()
	require.False(t, c, k)
}
