package bitvector

import (
	"testing"

	"github.com/aws/aws-sdk-go/service/ebs"
)

func int64Ptr(v int64) *int64 {
	return &v
}

func stringPtr(v string) *string {
	return &v
}

func TestCSICBCreate(t *testing.T) {
	mockEBSCBResponse := ebs.ListChangedBlocksOutput{
		ChangedBlocks: []*ebs.ChangedBlock{
			{
				BlockIndex:       int64Ptr(0),
				FirstBlockToken:  stringPtr("ACQBAY0q7BLrLGkS6o9WGTFX"),
				SecondBlockToken: stringPtr("ACQBAYdikelzxFNROFpr07n"),
			},
			{
				BlockIndex:       int64Ptr(1),
				SecondBlockToken: stringPtr("ACQBAcdXStq8R0oyKRAI7DH"),
			},
			{
				BlockIndex:       int64Ptr(2),
				SecondBlockToken: stringPtr("ACQBAbSQqHFig1bWpK5lSvY"),
			},
			{
				BlockIndex:       int64Ptr(3),
				SecondBlockToken: stringPtr("ACQBAe3l72sov46KUF/wKLC"),
			},
			{
				BlockIndex:       int64Ptr(4),
				SecondBlockToken: stringPtr("ACQBAZTV1a8szGlIhfj49Pe"),
			},
			{
				BlockIndex:       int64Ptr(5),
				SecondBlockToken: stringPtr("ACQBAQXx/YESkOWKFX/UrWn"),
			},
			{
				BlockIndex:       int64Ptr(6),
				SecondBlockToken: stringPtr("ACQBAYtwfA/txSVQK3/+tUi"),
			},
			{
				BlockIndex:       int64Ptr(7),
				SecondBlockToken: stringPtr("ACQBAdgk4t8UXFpkTKw8M/G"),
			},
			{
				BlockIndex:       int64Ptr(8),
				SecondBlockToken: stringPtr("ACQBAb6ueB7JF0E1ruzWGHa"),
			},
			{
				BlockIndex:       int64Ptr(9),
				SecondBlockToken: stringPtr("ACQBAd9iYg130xvFtPdI7Ec"),
			},
			{
				BlockIndex:       int64Ptr(10),
				SecondBlockToken: stringPtr("ACQBAfybjiA9kzj1EwR0b3O"),
			},
			{
				BlockIndex:       int64Ptr(11),
				SecondBlockToken: stringPtr("ACQBAeSPrf90/TemVsUGvYd"),
			},
			{
				BlockIndex:       int64Ptr(12),
				SecondBlockToken: stringPtr("ACQBAeEnUrWaX0W4p+sxntz"),
			},
			{
				BlockIndex:       int64Ptr(13),
				SecondBlockToken: stringPtr("ACQBAd6Brymqb4zO1qZFApN"),
			},
			{
				BlockIndex:       int64Ptr(15),
				SecondBlockToken: stringPtr("ACQBAQjPwaArxapWdE08N8J"),
			},
			{
				BlockIndex:       int64Ptr(16),
				SecondBlockToken: stringPtr("ACQBATbVeGwkwh1KES1CFxt"),
			},
			{
				BlockIndex:       int64Ptr(17),
				SecondBlockToken: stringPtr("ACQBAUVaPadKApURMfbx1qa"),
			},
			{
				BlockIndex:       int64Ptr(18),
				SecondBlockToken: stringPtr("ACQBAc6doy23SwGWtnJCP4L"),
			},
			{
				BlockIndex:       int64Ptr(19),
				SecondBlockToken: stringPtr("ACQBAYtneArS7TJdEQvEKdH"),
			},
			{
				BlockIndex:       int64Ptr(20),
				SecondBlockToken: stringPtr("ACQBAUCOcZucXBnYuyT0zkB"),
			},
			{
				BlockIndex:       int64Ptr(21),
				SecondBlockToken: stringPtr("ACQBAStQ+T51MpPNCQrLN9V"),
			},
			{
				BlockIndex:       int64Ptr(22),
				SecondBlockToken: stringPtr("ACQBAdS9v+EDSn34rPTipV+"),
			},
			{
				BlockIndex:       int64Ptr(23),
				SecondBlockToken: stringPtr("ACQBAYsXOGV8dyk4FBZiDSn"),
			},
			{
				BlockIndex:       int64Ptr(24),
				SecondBlockToken: stringPtr("ACQBAcBiGUUyj4+kXcwlFhB"),
			},
			{
				BlockIndex:       int64Ptr(25),
				SecondBlockToken: stringPtr("ACQBAVLYK6BRENwinUjmTdC"),
			},
			{
				BlockIndex:       int64Ptr(26),
				SecondBlockToken: stringPtr("ACQBAUf4/CnLNfJWLpUxAbY"),
			},
			{
				BlockIndex:       int64Ptr(27),
				SecondBlockToken: stringPtr("ACQBAQDsTbR6wDc0L+UE1S3"),
			},
			{
				BlockIndex:       int64Ptr(29),
				SecondBlockToken: stringPtr("ACQBAeJBNAFvJ0mXZyI6vA9"),
			},
			{
				BlockIndex:       int64Ptr(800),
				SecondBlockToken: stringPtr("ACQBAaxAUlbkn/Er/0BTtVs"),
			},
			{
				BlockIndex:       int64Ptr(802),
				SecondBlockToken: stringPtr("ACQBAaxAUlbkn/Er/0BTtVs"),
			},
		},
		//ExpiryTime: "2022-11-28T21:59:26.776000+05:30",
		VolumeSize: int64Ptr(4),
		BlockSize:  int64Ptr(524288),
	}
	bv := NewChangedBlocksBitVectorFromEBS(mockEBSCBResponse)
	b, err := bv.Serialize()
	if err != nil {
		t.Error(err)
	}
	_, err = Deserialize(b)
	if err != nil {
		t.Error(err)
	}
}
