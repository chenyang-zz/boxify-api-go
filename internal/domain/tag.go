package domain

type TagScope string

const (
	TagScopeAll      TagScope = "all"
	TagScopeDocument TagScope = "document"
	TagScopeImage    TagScope = "image"
)
