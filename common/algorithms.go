package common

func ContainsAll(requiredStrings []string, actualStrings []string) bool {
	for _, req := range requiredStrings {
		found := false
		for _, act := range actualStrings {
			if req == act {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}

func Contains(requiredStrings []string, actualString string) bool {
	for _, req := range requiredStrings {
		if req == actualString {
			return true
		}

	}
	return false
}
