package models

type Role string

const (
	RoleAdmin    Role = "ADMIN"
	RoleEditor   Role = "EDITOR"
	RoleReporter Role = "REPORTER"
	RoleUser     Role = "USER"
)

type ArticleStatus string

const (
	ArticleStatusDraft     ArticleStatus = "DRAFT"
	ArticleStatusPublished ArticleStatus = "PUBLISHED"
	ArticleStatusArchived  ArticleStatus = "ARCHIVED"
)
