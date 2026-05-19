package repository

import "gorm.io/gorm"

// NewRepos 返回一个*Repos，每个存储库上的调用都运行在自己的短自动提交事务中
// 对于必须是原子操作的多步骤写入，最好通过UnitOfWork.WithTx获得一个tx绑定的*Repos
func NewRepos(db *gorm.DB) *Repos {
	return &Repos{
		User:        NewUserRepo(db),
		File:        NewFileRepo(db),
		FileCopy:    NewFileCopyRepo(db),
		Message:     NewMessageRepo(db),
		Maintenance: NewMaintenanceRepo(db),
	}
}

// GormUnitOfWork implements UnitOfWork on top of *gorm.DB's transaction support.
type GormUnitOfWork struct {
	db *gorm.DB
}

// NewGormUnitOfWork creates a GormUnitOfWork bound to the root *gorm.DB.
func NewGormUnitOfWork(db *gorm.DB) *GormUnitOfWork {
	return &GormUnitOfWork{db: db}
}

// 开启事务，返回nil则提交，否则回滚
func (u *GormUnitOfWork) WithTx(fn func(repos *Repos) error) error {
	return u.db.Transaction(func(tx *gorm.DB) error {
		return fn(NewRepos(tx))
	})
}
