package model

import "github.com/hmans/beans/internal/githistory"

// BeanCommit is a commit in a bean's git history. It remembers the bean's
// file so the diff can be resolved on demand.
type BeanCommit struct {
	githistory.Commit
	File githistory.File
}
