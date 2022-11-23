package bitvector

import (
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go/service/ebs"
	"github.com/dropbox/godropbox/container/bitvector"
)

type Response struct {
	BitVector       []byte `json:"bitVector"`
	BlockSizeBytes  int    `json:"blockSize"`
	VolumeSizeBytes int    `json:"volumeSizeBytes"`
	TotalBlocks     int    `json:"totalBlocks"`
}

type ChangedBlocks struct {
	BV *bitvector.BitVector

	// TODO: Handle pagination
	BlockSizeBytes  int
	VolumeSizeBytes int
	TotalBlocks     int
}

func NewChangedBlocksBitVectorFromEBS(cbo ebs.ListChangedBlocksOutput) *ChangedBlocks {
	volumeSizeBytes := *cbo.VolumeSize * 107374182 // Convert GB to bytes
	totalBlocks := (volumeSizeBytes + *cbo.BlockSize) / *cbo.BlockSize
	// BitVector size = number of blocks
	// Size required to store 1block=1bit
	// Bytes required = totalBlocks/8
	fmt.Println("Total blocks:", totalBlocks)
	v := bitvector.NewBitVector(make([]byte, (totalBlocks+8)/8), int(totalBlocks)) // no of bytes = no of blocks/8 (1 bit for each ChangedBlocks)
	for _, cbData := range cbo.ChangedBlocks {
		v.Set(1, int(*cbData.BlockIndex)) // Check type convertion safety
	}

	// Debug: Print bitmap
	fmt.Println("Changed Blocks map::")
	//blockBytes := cb.Bytes()
	for i := 0; i < v.Length(); i++ {
		if v.Element(i) != 0 {
			fmt.Printf("x ")
			continue
		}
		fmt.Printf("0 ")
	}
	fmt.Println("")
	return &ChangedBlocks{
		BV:              v,
		BlockSizeBytes:  int(*cbo.BlockSize),
		VolumeSizeBytes: int(*cbo.VolumeSize),
		TotalBlocks:     int(totalBlocks),
	}
}

func (cb *ChangedBlocks) Serialize() ([]byte, error) {
	resp := Response{
		BitVector:       cb.BV.Bytes(),
		BlockSizeBytes:  cb.BlockSizeBytes,
		VolumeSizeBytes: cb.VolumeSizeBytes,
		TotalBlocks:     cb.TotalBlocks,
	}
	fmt.Println("Encoded BitVector:", resp.BitVector)
	return json.Marshal(resp)
}

func Deserialize(data []byte) (*ChangedBlocks, error) {
	var cbResp Response
	err := json.Unmarshal(data, &cbResp)
	if err != nil {
		return nil, err
	}
	fmt.Println("Deserialized BitVector:", cbResp.BitVector)

	// BitVector size = number of blocks
	// Size required to store 1bit=1block
	// Bytes required = totalBlocks/8
	fmt.Println("Total blocks:", cbResp.TotalBlocks)
	v := bitvector.NewBitVector(cbResp.BitVector, cbResp.TotalBlocks)
	cb := &ChangedBlocks{
		BV:              v,
		BlockSizeBytes:  cbResp.BlockSizeBytes,
		VolumeSizeBytes: cbResp.VolumeSizeBytes,
	}

	// Debug: Print bitmap
	fmt.Println("Changed Blocks map::")
	//blockBytes := cb.Bytes()
	for i := 0; i < cb.BV.Length(); i++ {
		if cb.BV.Element(i) != 0 {
			fmt.Printf("x ")
			continue
		}
		fmt.Printf("0 ")
	}
	return cb, err
}
