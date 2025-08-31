package avail

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ✅ Test round-trip of BlobPointer marshal/unmarshal
func TestBlobPointerRoundTrip(t *testing.T) {
	original := NewBlobPointer(12345, 7, [32]byte{1, 2, 3})
	bytes, err := original.MarshalToBinary()
	require.NoError(t, err)

	var decoded BlobPointer
	err = decoded.UnmarshalFromBinary(bytes)
	require.NoError(t, err)

	assert.Equal(t, *original, decoded)
}

// ✅ Test envelope pack/unpack with BlobPointer
func TestPackUnpackEnvelope(t *testing.T) {
	original := []byte("hello world")
	envelope, err := PackEnvelopeWithMsgType(DAM_TYPE_BLOB_POINTER, original)
	require.NoError(t, err)

	msgType, payload, err := UnpackEnvelopeForMsgType(envelope)
	require.NoError(t, err)

	assert.Equal(t, uint8(DAM_TYPE_BLOB_POINTER), msgType)
	assert.Equal(t, original, payload)
}

// ✅ Test MerkleProofInput round-trip
func TestMerkleProofInputRoundTrip(t *testing.T) {
	original := MerkleProofInput{
		DataRootProof: [][32]byte{
			{173, 50, 40, 182, 118, 247, 211, 205, 66, 132, 165, 68, 63, 23, 241, 150, 43, 54, 228, 145, 179, 10, 64, 178, 64, 88, 73, 229, 151, 186, 95, 181},
			{81, 248, 78, 114, 121, 205, 246, 172, 184, 26, 247, 122, 236, 100, 246, 24, 247, 16, 41, 183, 217, 198, 211, 124, 3, 92, 55, 19, 78, 81, 122, 242},
			{105, 200, 69, 141, 214, 45, 39, 234, 154, 189, 64, 88, 108, 229, 62, 82, 32, 212, 59, 98, 108, 39, 247, 100, 104, 165, 126, 148, 52, 127, 13, 107},
			{90, 2, 30, 101, 234, 92, 107, 118, 70, 155, 104, 219, 40, 199, 163, 144, 131, 110, 34, 194, 28, 111, 149, 205, 239, 77, 52, 8, 235, 107, 128, 80},
			{161, 66, 157, 58, 135, 235, 114, 86, 237, 149, 23, 84, 136, 117, 99, 160, 214, 72, 96, 196, 136, 84, 26, 110, 157, 79, 171, 38, 135, 227, 6, 88},
			{153, 192, 154, 157, 133, 237, 44, 209, 222, 10, 185, 2, 105, 111, 216, 21, 5, 111, 194, 157, 75, 78, 12, 35, 201, 13, 85, 176, 215, 158, 165, 212},
			{252, 225, 122, 81, 66, 13, 91, 136, 167, 217, 26, 113, 105, 115, 101, 74, 156, 70, 52, 87, 251, 31, 177, 227, 246, 41, 119, 71, 58, 220, 71, 230},
			{233, 226, 152, 144, 241, 188, 123, 237, 78, 185, 34, 25, 84, 239, 42, 78, 171, 75, 105, 167, 49, 1, 130, 135, 124, 242, 184, 189, 178, 200, 245, 28},
			{88, 91, 133, 61, 102, 164, 187, 200, 166, 83, 185, 7, 241, 98, 72, 83, 162, 54, 165, 33, 100, 138, 34, 86, 171, 109, 167, 239, 210, 171, 67, 97},
			{80, 109, 134, 88, 45, 37, 36, 5, 184, 64, 1, 135, 146, 202, 210, 191, 18, 89, 241, 239, 90, 165, 248, 135, 225, 60, 178, 240, 9, 79, 81, 225},
			{255, 255, 10, 215, 230, 89, 119, 47, 149, 52, 193, 149, 200, 21, 239, 196, 1, 78, 241, 225, 218, 237, 68, 4, 192, 99, 133, 209, 17, 146, 233, 43},
		},
		LeafProof:     [][32]byte{}, // Empty
		RangeHash:     [32]byte{218, 51, 52, 143, 216, 175, 43, 27, 235, 186, 222, 186, 213, 219, 94, 198, 195, 197, 45, 79, 57, 244, 190, 143, 134, 173, 120, 171, 103, 106, 125, 226},
		DataRootIndex: big.NewInt(143),
		BlobRoot:      [32]byte{63, 109, 5, 38, 27, 218, 88, 55, 255, 136, 45, 122, 132, 99, 67, 15, 186, 205, 124, 227, 52, 227, 9, 212, 73, 236, 182, 45, 72, 172, 209, 67},
		BridgeRoot:    [32]byte{}, // Empty
		Leaf:          [32]byte{142, 204, 27, 154, 88, 162, 236, 187, 225, 40, 188, 34, 193, 120, 141, 171, 109, 118, 85, 107, 113, 166, 157, 42, 221, 12, 250, 89, 34, 159, 50, 172},
		LeafIndex:     big.NewInt(0),
	}

	encoded, err := original.EnodeToBinary()
	require.NoError(t, err)

	var decoded MerkleProofInput
	err = decoded.DecodeFromBinary(encoded)
	require.NoError(t, err)

	assert.Equal(t, original.DataRootProof, decoded.DataRootProof)
	assert.Equal(t, original.BlobRoot, decoded.BlobRoot)
	assert.Equal(t, original.Leaf, decoded.Leaf)

	assert.Equal(t, 0, original.DataRootIndex.Cmp(decoded.DataRootIndex))
	assert.Equal(t, 0, original.LeafIndex.Cmp(decoded.LeafIndex))

}

// ✅ Test invalid envelope type
func TestUnpackEnvelopeInvalid(t *testing.T) {
	_, _, err := UnpackEnvelopeForMsgType([]byte{0x99, 0x01, 0x02})
	assert.Error(t, err, "should error for invalid msg type")
}
