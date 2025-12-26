package deps

// FindLanguage returns the Language with the given name from the provided list, or nil if not found.
func FindLanguage(name string, languages []*Language) *Language {
	for _, lang := range languages {
		if lang.Name == name {
			return lang
		}
	}
	return nil
}
