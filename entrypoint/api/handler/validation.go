package handler

import "regexp"

var (
	// alphanumericRequiredRegex 英数字1文字以上のパターン（必須フィールド用）
	alphanumericRequiredRegex = regexp.MustCompile(`^[a-zA-Z0-9]+$`)

	// alphanumericOptionalRegex 英数字0文字以上のパターン（任意フィールド用）
	alphanumericOptionalRegex = regexp.MustCompile(`^[a-zA-Z0-9]*$`)
)
