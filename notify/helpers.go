package notify

import "strings"

func hideString(phoneNumber string, markLen int) string {
	if phoneNumber == "" {
		return ""
	}

	if len(phoneNumber) < markLen*2 {
		return strings.Repeat("*", len(phoneNumber))
	}

	start := phoneNumber[len(phoneNumber)-markLen:]
	end := phoneNumber[0:markLen]

	result := strings.ReplaceAll(phoneNumber, start, strings.Repeat("*", markLen))
	result = strings.ReplaceAll(result, end, strings.Repeat("*", markLen))

	return result
}
