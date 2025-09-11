package prompt

import (
	_ "embed"
	"fmt"
)

//go:embed translate_prompt.md
var translatePrompt string

func TranslatePrompt(fromLang, toLang string) string {
	return fmt.Sprintf(translatePrompt, fromLang, toLang, toLang, toLang)
}
