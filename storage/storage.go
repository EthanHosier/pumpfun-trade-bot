package storage

import (
	"errors"

	"github.com/nedpals/supabase-go"
)

type Storage interface {
	Store(table DbTableName, data interface{}) (interface{}, error)
	StoreAll(table DbTableName, data []interface{}) (interface{}, error)
	Get(table DbTableName, id string) (interface{}, error)
}

type DbTableName string

const (
	DbCoinsTable  DbTableName = "coins"
	DbTradesTable DbTableName = "trades"
)

var (
	tableKeyMap = map[DbTableName]string{
		DbCoinsTable:  "mint",
		DbTradesTable: "id",
	}
)

type SupabaseStorage struct {
	client *supabase.Client
}

func NewSupabaseStorage(url string, serviceKey string) *SupabaseStorage {
	return &SupabaseStorage{client: supabase.CreateClient(url, serviceKey)}
}

func (s *SupabaseStorage) Store(table DbTableName, data interface{}) (interface{}, error) {
	var results []interface{}
	err := s.client.DB.From(string(table)).Insert(data).Execute(&results)

	return results, err
}

func (s *SupabaseStorage) StoreAll(table DbTableName, data []interface{}) (interface{}, error) {
	var results []interface{}
	err := s.client.DB.From(string(table)).Insert(data).Execute(&results)

	return results, err
}

func (s *SupabaseStorage) Get(table DbTableName, id string) (interface{}, error) {
	var result []interface{}
	err := s.client.DB.From(string(table)).Select("*").Limit(1).Eq(tableKeyMap[table], id).Execute(&result)

	if err != nil {
		return nil, err
	}

	if len(result) == 0 {
		return nil, errors.New("not found")
	}

	return result[0], nil
}
