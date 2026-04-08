package repo

import "stock/dal/dao"

type JiuyanRawDataRepository interface {
	dao.JiuyanRawDataDAO
}

type jiuyanRawDataRepoImpl struct {
	dao.JiuyanRawDataDAO
}

func NewJiuyanRawDataRepository() JiuyanRawDataRepository {
	return &jiuyanRawDataRepoImpl{dao.NewJiuyanRawDataDAO()}
}
