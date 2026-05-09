package gameops

import "strconv"

func strconvFormatInt(value int64) string {
	return strconv.FormatInt(value, 10)
}

func strconvParseInt(value string) (int64, error) {
	return strconv.ParseInt(value, 10, 64)
}
