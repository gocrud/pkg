package seatax

// firstXID 返回第一个非空的值,用于在主备 XID 头名之间兜底取值。
func firstXID(primary, fallback string) string {
	if primary != "" {
		return primary
	}
	return fallback
}
