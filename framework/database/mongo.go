package database

import (
	"context"
	"reflect"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type Document struct {
	ObjectId string `json:"objectId,omitempty" bson:"objectId,omitempty" validate:"len=20"`
}

type Connection struct {
	cli         *mongo.Client
	database    *mongo.Database              // mongo default database
	databases   map[string]*mongo.Database   // mongo database instance
	collections map[string]*mongo.Collection // mongo collection instance
	timeout     time.Duration                // default timeout
}

type Collection struct {
	dname      []string
	cname      string
	alloc      func() any
	collection *mongo.Collection
}

var conn Connection

func ConnectMongo(connectString string, timeout time.Duration) (err error) {
	opt := options.Client().ApplyURI(connectString)
	opt.SetMaxConnecting(32)
	opt.SetMaxPoolSize(64)
	opt.SetRetryReads(false)

	conn.cli, err = mongo.NewClient(opt)
	if err != nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	err = conn.cli.Connect(ctx)
	if err != nil {
		return
	}

	//ping the database
	err = conn.cli.Ping(ctx, nil)
	if err != nil {
		panic(err)
	}

	conn.databases = make(map[string]*mongo.Database)
	conn.collections = make(map[string]*mongo.Collection)
	conn.timeout = timeout
	return
}

func SetDatabase(name string, isDefault bool) {
	conn.databases[name] = conn.cli.Database(name)
	if isDefault {
		conn.database = conn.cli.Database(name)
	}

	if conn.database == nil {
		conn.database = conn.cli.Database(name)
	}
}

func GetDatabase(name string) *mongo.Database {
	if c, ok := conn.databases[name]; ok {
		return c
	}

	database := conn.cli.Database(name)
	conn.databases[name] = database
	return database
}

func Check() bool {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	client := conn.cli

	if err := client.Ping(ctx, nil); err != nil {
		return false
	}

	return true
}

func GetCollection(name string, d ...string) *mongo.Collection {
	if conn.cli == nil {
		log.Debug("conn.cli is nil")
		return nil
	}

	if c, ok := conn.collections[name]; ok {
		return c
	}

	var collection *mongo.Collection
	if len(d) == 0 {
		collection = conn.database.Collection(name)
	} else {
		database, ok := conn.databases[d[0]]
		if ok {
			collection = database.Collection(name)
		} else {
			log.Error("cannot get database ", d[0])
		}
	}

	if collection == nil {
		return nil
	}

	log.Debug("append new collection '", name, "' database = ", d)
	conn.collections[name] = collection
	return collection
}

func NewCollection[T any](name string, database ...string) Collection {
	allocateElement := func() any {
		return new(T)
	}

	warpper := Collection{
		dname:      database,
		cname:      name,
		alloc:      allocateElement,
		collection: nil,
	}

	return warpper
}

func Projection(fields []string) bson.M {
	if len(fields) == 0 {
		return nil
	}

	projection := make(bson.M, len(fields))
	for _, v := range fields {
		projection[v] = true
	}

	return projection
}

func Ordered(fields []string) bson.M {
	if len(fields) == 0 {
		return nil
	}

	orderfields := make(bson.M, len(fields))
	for _, v := range fields {
		orderfields[v] = true
	}

	return orderfields
}

func (c *Collection) Get() *mongo.Collection {
	if c.collection == nil {
		collection := GetCollection(c.cname, c.dname...)
		if collection == nil {
			return nil
		}

		c.collection = collection
	}

	return c.collection
}

func (c *Collection) Insert(doc any) error {
	ctx, cancel := context.WithTimeout(context.Background(), conn.timeout)
	defer cancel()

	return c.InsertCtx(ctx, doc)
}

func (c *Collection) InsertCtx(ctx context.Context, doc any) error {
	_, err := c.Get().InsertOne(ctx, doc)
	if err != nil {
		return err
	}

	return nil
}

func (c *Collection) InsertMany(doc []any) error {
	ctx, cancel := context.WithTimeout(context.Background(), conn.timeout)
	defer cancel()

	return c.InsertManyCtx(ctx, doc)
}

func (c *Collection) InsertManyCtx(ctx context.Context, doc []any) error {
	_, err := c.Get().InsertMany(ctx, doc)
	if err != nil {
		return err
	}

	return nil
}

func (c *Collection) DeleteOne(filter any) error {
	ctx, cancel := context.WithTimeout(context.Background(), conn.timeout)
	defer cancel()

	return c.DeleteOneCtx(ctx, filter)
}

func (c *Collection) DeleteOneCtx(ctx context.Context, filter any) error {
	_, err := c.Get().DeleteOne(ctx, filter)
	return err
}

func (c *Collection) DeleteMany(filter any) (deleted int64, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), conn.timeout)
	defer cancel()

	return c.DeleteManyCtx(ctx, filter)
}

func (c *Collection) DeleteManyCtx(ctx context.Context, filter any) (deleted int64, err error) {
	res, err := c.Get().DeleteMany(ctx, filter)

	return res.DeletedCount, err
}

func (c Collection) Update(filter any, update any, upsert bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), conn.timeout)
	defer cancel()

	return c.UpdateCtx(ctx, filter, update, upsert)
}

func (c Collection) UpdateCtx(ctx context.Context, filter any, update any, upsert bool) error {
	opt := options.Update()
	opt.SetUpsert(upsert)

	_, err := c.Get().UpdateMany(ctx, filter, update, opt)

	if err != nil {
		return err
	}

	return nil
}

func (c Collection) UpdateOne(filter any, update any, upsert bool) error {
	ctx, cancel := context.WithTimeout(context.Background(), conn.timeout)
	defer cancel()

	return c.UpdateOneCtx(ctx, filter, update, upsert)
}

func (c Collection) UpdateOneCtx(ctx context.Context, filter any, update any, upsert bool) error {
	opt := options.Update()
	opt.SetUpsert(upsert)
	_, err := c.Get().UpdateOne(ctx, filter, update, opt)

	if err != nil {
		return err
	}

	return nil
}

func (c *Collection) CountDocuments(filter any) int64 {
	ctx, cancel := context.WithTimeout(context.Background(), conn.timeout)
	defer cancel()

	return c.CountDocumentsCtx(ctx, filter)
}

func (c *Collection) CountDocumentsCtx(ctx context.Context, filter any) int64 {
	res, err := c.Get().CountDocuments(ctx, filter)
	if err != nil {
		return 0
	}

	return res
}

func (c *Collection) FindOne(filter any, fields ...string) (result any, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), conn.timeout)
	defer cancel()

	return c.FindOneCtx(ctx, filter, fields...)
}

func (c *Collection) FindOneCtx(ctx context.Context, filter any, fields ...string) (result any, err error) {
	opt := options.FindOne().SetProjection(Projection(fields))
	res := c.Get().FindOne(ctx, filter, opt)

	ele := c.alloc()
	err = res.Decode(ele)
	return reflect.ValueOf(ele).Elem().Interface(), err
}

func (c *Collection) Find(filter bson.M, fields ...string) (result []any, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), conn.timeout)
	defer cancel()

	return c.FindCtx(ctx, filter, fields...)
}

func (c *Collection) FindCtx(ctx context.Context, filter bson.M, fields ...string) (result []any, err error) {
	opt := options.Find().SetProjection(Projection(fields))
	var cur *mongo.Cursor
	if cur, err = c.Get().Find(ctx, filter, opt); err != nil {
		return
	} else {
		for cur.Next(ctx) {
			ele := c.alloc()
			if err = cur.Decode(ele); err != nil {
				return
			}

			result = append(result, reflect.ValueOf(ele).Elem().Interface())
		}
	}

	return
}

func (c *Collection) Query(ctx context.Context, filter bson.M, skip int64, limit int64, orders []string, fields ...string) (result []any, total int64, err error) {
	opt := options.Find()
	opt.SetProjection(Projection(fields))
	opt.SetSkip(skip)
	opt.SetLimit(limit)
	opt.SetSort(Ordered(orders))
	var cur *mongo.Cursor
	if cur, err = c.Get().Find(ctx, filter, opt); err != nil {
		return
	} else {
		for cur.Next(ctx) {
			ele := c.alloc()
			if err = cur.Decode(ele); err != nil {
				return
			}

			result = append(result, reflect.ValueOf(ele).Elem().Interface())
		}

		total = c.CountDocumentsCtx(ctx, filter)
	}

	return
}

func (c *Collection) FindOneAndUpdate(filter any, update any, fields ...string) (result any, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), conn.timeout)
	defer cancel()

	return c.FindOneAndUpdateCtx(ctx, filter, update, fields...)
}

func (c *Collection) FindOneAndUpdateCtx(ctx context.Context, filter any, update any, fields ...string) (result any, err error) {
	var opt = options.FindOneAndUpdate()

	if len(fields) > 0 {
		opt.SetProjection(Projection(fields))
	}

	res := c.Get().FindOneAndUpdate(ctx, filter, update, opt)
	ele := c.alloc()
	err = res.Decode(ele)
	return reflect.ValueOf(ele).Elem().Interface(), err
}

func (c *Collection) BulkWrite(models []mongo.WriteModel, ordered bool) (modified int64, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), conn.timeout)
	defer cancel()

	return c.BulkWriteCtx(ctx, models, ordered)
}

func (c *Collection) BulkWriteCtx(ctx context.Context, models []mongo.WriteModel, ordered bool) (modified int64, err error) {
	opts := options.BulkWrite().SetOrdered(ordered)
	res, err := c.Get().BulkWrite(ctx, models, opts)
	return res.ModifiedCount, err
}
