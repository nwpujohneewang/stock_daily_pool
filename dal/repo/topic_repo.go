package repo

import (
	"context"
	"stock/dal/dao"
	"stock/model/dal_model"
)

type TopicRepository interface {
	GetByID(ctx context.Context, id int64) (*dal_model.Topic, error)
	GetByIDs(ctx context.Context, ids []int64) ([]dal_model.Topic, error)
	GetByName(ctx context.Context, name string) (*dal_model.Topic, error)
	GetByNames(ctx context.Context, names []string) ([]dal_model.Topic, error)
	Upsert(ctx context.Context, topic *dal_model.Topic) error
	UpsertBatch(ctx context.Context, topics []dal_model.Topic) error
	List(ctx context.Context, keyword string, page, pageSize int) ([]dal_model.Topic, int64, error)
	GetActiveTopics(ctx context.Context) ([]dal_model.Topic, error)
	GetIDByName(ctx context.Context, name string) (int64, error)
	Update(ctx context.Context, id int64, name string, isActive bool, priority int) error
	Delete(ctx context.Context, id int64) error
	Merge(ctx context.Context, sourceID, targetID int64) error
	GetNamesByIDs(ctx context.Context, ids []int64) (map[int64]string, error)
}

type topicRepoImpl struct {
	dao dao.TopicDAO
}

func NewTopicRepository() TopicRepository {
	return &topicRepoImpl{dao: dao.NewTopicDAO()}
}

func (r *topicRepoImpl) GetByID(ctx context.Context, id int64) (*dal_model.Topic, error) {
	return r.dao.GetByID(ctx, id)
}

func (r *topicRepoImpl) GetByIDs(ctx context.Context, ids []int64) ([]dal_model.Topic, error) {
	return r.dao.GetByIDs(ctx, ids)
}

func (r *topicRepoImpl) GetByName(ctx context.Context, name string) (*dal_model.Topic, error) {
	return r.dao.GetByName(ctx, name)
}

func (r *topicRepoImpl) GetByNames(ctx context.Context, names []string) ([]dal_model.Topic, error) {
	return r.dao.GetByNames(ctx, names)
}

func (r *topicRepoImpl) Upsert(ctx context.Context, topic *dal_model.Topic) error {
	return r.dao.Upsert(ctx, topic)
}

func (r *topicRepoImpl) UpsertBatch(ctx context.Context, topics []dal_model.Topic) error {
	return r.dao.UpsertBatch(ctx, topics)
}

func (r *topicRepoImpl) List(ctx context.Context, keyword string, page, pageSize int) ([]dal_model.Topic, int64, error) {
	return r.dao.List(ctx, keyword, page, pageSize)
}

func (r *topicRepoImpl) GetActiveTopics(ctx context.Context) ([]dal_model.Topic, error) {
	return r.dao.GetActiveTopics(ctx)
}

func (r *topicRepoImpl) GetIDByName(ctx context.Context, name string) (int64, error) {
	return r.dao.GetIDByName(ctx, name)
}

func (r *topicRepoImpl) Update(ctx context.Context, id int64, name string, isActive bool, priority int) error {
	return r.dao.Update(ctx, id, name, isActive, priority)
}

func (r *topicRepoImpl) Delete(ctx context.Context, id int64) error {
	return r.dao.Delete(ctx, id)
}

func (r *topicRepoImpl) Merge(ctx context.Context, sourceID, targetID int64) error {
	return r.dao.Merge(ctx, sourceID, targetID)
}

func (r *topicRepoImpl) GetNamesByIDs(ctx context.Context, ids []int64) (map[int64]string, error) {
	return r.dao.GetNamesByIDs(ctx, ids)
}
