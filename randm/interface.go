package randm

// Method describe rand tool provide methodds
type Method interface {
	// Generate to generate a RID
	// RID has a series sub series methods is based on RID
	GenerateRID() RID

	// IsValidateRID for quick check is input RID is avalidate RID
	IsValidateRID(rid int64) (res bool)

	// GenRandomString by given length
	// NOTE: this function is pure generate rand string,
	// it can not revert to a validated RID
	GenRandomString(length uint) (resStr string)

	// GetNodeFromRID parses the RID to extract the Node ID
	GetNodeFromRID(rid int64) int64
}
