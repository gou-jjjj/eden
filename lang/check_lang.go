package lang

import "unicode"

// 语言代码常量
const (
	ZH = "Chinese"  // 中文
	EN = "English"  // 英文
	JA = "Japanese" // 日文
	KO = "Korean"   // 韩文
	RU = "Russian"  // 俄文
	AR = "Arabic"   // 阿拉伯文
	EL = "Greek"    // 希腊文
)

// LanguageChecker 语言检查器接口
type LanguageChecker interface {
	Check(s string) bool
	Name() string
}

// 基础检查函数
func isInRangeTable(content string, rangeTab *unicode.RangeTable) bool {
	for _, word := range content {
		if unicode.Is(rangeTab, word) {
			return true
		}
	}
	return false
}

// 中文检查器
type ChineseChecker struct{}

func (c ChineseChecker) Check(s string) bool {
	return isInRangeTable(s, unicode.Han)
}

func (c ChineseChecker) Name() string {
	return ZH
}

// 英文检查器
type EnglishChecker struct{}

func (c EnglishChecker) Check(s string) bool {
	return isInRangeTable(s, unicode.Latin)
}

func (c EnglishChecker) Name() string {
	return EN
}

// 日文检查器
type JapaneseChecker struct{}

func (c JapaneseChecker) Check(s string) bool {
	// 日文主要包含 Hiragana（平假名）, Katakana（片假名）, 以及部分汉字
	return isInRangeTable(s, unicode.Hiragana) ||
		isInRangeTable(s, unicode.Katakana) ||
		isInRangeTable(s, unicode.Han)
}

func (c JapaneseChecker) Name() string {
	return JA
}

// 韩文检查器
type KoreanChecker struct{}

func (c KoreanChecker) Check(s string) bool {
	// 韩文：Hangul
	return isInRangeTable(s, unicode.Hangul)
}

func (c KoreanChecker) Name() string {
	return KO
}

// 俄文检查器
type RussianChecker struct{}

func (c RussianChecker) Check(s string) bool {
	return isInRangeTable(s, unicode.Cyrillic)
}

func (c RussianChecker) Name() string {
	return RU
}

// 阿拉伯文检查器
type ArabicChecker struct{}

func (c ArabicChecker) Check(s string) bool {
	return isInRangeTable(s, unicode.Arabic)
}

func (c ArabicChecker) Name() string {
	return AR
}

// 希腊文检查器
type GreekChecker struct{}

func (c GreekChecker) Check(s string) bool {
	return isInRangeTable(s, unicode.Greek)
}

func (c GreekChecker) Name() string {
	return EL
}

// 语言检查器映射
var LangMapChecks = map[string]LanguageChecker{
	ZH: ChineseChecker{},
	EN: EnglishChecker{},
	JA: JapaneseChecker{},
	KO: KoreanChecker{},
	RU: RussianChecker{},
	AR: ArabicChecker{},
	EL: GreekChecker{},
}

// 辅助函数：检测文本语言
func DetectLanguage(text string) (string, string) {
	for code, checker := range LangMapChecks {
		if checker.Check(text) {
			return code, checker.Name()
		}
	}
	return "", "Unknown"
}

// 辅助函数：检查文本是否为特定语言
func IsLanguage(text, langCode string) bool {
	checker, exists := LangMapChecks[langCode]
	if !exists {
		return false
	}
	return checker.Check(text)
}
