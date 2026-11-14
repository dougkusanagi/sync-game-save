package core

type Action int

const (
	Nothing Action = iota
	RestoreFromCloud
	UploadLocal
	Conflict
)

func Decide(base *string, local Index, cloud *string) Action {
	digest := local.Digest()
	localChanged := len(local.Files) != 0
	if base != nil {
		localChanged = *base != digest
	}
	cloudChanged := (cloud == nil) != (base == nil) || (cloud != nil && base != nil && *cloud != *base)
	switch {
	case !localChanged && !cloudChanged:
		return Nothing
	case !localChanged:
		return RestoreFromCloud
	case !cloudChanged:
		return UploadLocal
	case cloud != nil && *cloud == digest:
		return Nothing
	default:
		return Conflict
	}
}
