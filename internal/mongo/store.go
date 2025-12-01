package mongo

import (
    "context"
    "encoding/json"
    "fmt"
    "io"
    "os"
    "time"

    "go.mongodb.org/mongo-driver/bson"
    "go.mongodb.org/mongo-driver/mongo"
    "go.mongodb.org/mongo-driver/mongo/options"

    "github.com/redis/go-redis/v9"
    "go.uber.org/zap"
)

type Store struct {
    client *mongo.Client
    db     *mongo.Database
    rdb    *redis.Client
    ctx    context.Context
    log    *zap.SugaredLogger
}

func New(mongoURI, dbName, redisAddr string, sugar *zap.SugaredLogger) (*Store, error) {
    ctx := context.Background()
    clientOpts := options.Client().ApplyURI(mongoURI)
    client, err := mongo.Connect(ctx, clientOpts)
    if err != nil {
        if sugar != nil {
            sugar.Errorw("mongo connect failed", "error", err)
        }
        return nil, err
    }
    if err := client.Ping(ctx, nil); err != nil {
        if sugar != nil {
            sugar.Errorw("mongo ping failed", "error", err)
        }
        return nil, err
    }

    var rdb *redis.Client
    if redisAddr != "" {
        rdb = redis.NewClient(&redis.Options{Addr: redisAddr})
        // ping redis, but treat redis as optional: log a warning and continue if unreachable
        if err := rdb.Ping(ctx).Err(); err != nil {
            // do not fail the whole store initialization if redis is not ready; make it optional
            if sugar != nil {
                sugar.Warnw("redis ping failed; continuing without redis caching", "error", err)
            }
            rdb = nil
        }
    }

    s := &Store{client: client, db: client.Database(dbName), rdb: rdb, ctx: ctx, log: sugar}

    // Ensure index on idMeal
    coll := s.db.Collection("meals")
    _, _ = coll.Indexes().CreateOne(ctx, mongo.IndexModel{
        Keys: bson.D{{Key: "idMeal", Value: 1}},
        Options: options.Index().SetUnique(true).SetBackground(true),
    })

    return s, nil
}

// ImportFromFile reads a local all_meals.json and upserts into mongo. Idempotent.
func (s *Store) ImportFromFile(path string) error {
    f, err := os.Open(path)
    if err != nil {
        return fmt.Errorf("open file: %w", err)
    }
    defer f.Close()

    b, err := io.ReadAll(f)
    if err != nil {
        return fmt.Errorf("read file: %w", err)
    }

    var raw []map[string]interface{}
    if err := json.Unmarshal(b, &raw); err != nil {
        return fmt.Errorf("unmarshal: %w", err)
    }

    coll := s.db.Collection("meals")

    for _, m := range raw {
        id, _ := m["idMeal"].(string)
        if id == "" {
            continue
        }

        // build ingredients array
        var ingredients []bson.M
        for i := 1; i <= 20; i++ {
            inKey := fmt.Sprintf("strIngredient%d", i)
            msKey := fmt.Sprintf("strMeasure%d", i)
            inVal, _ := toString(m[inKey])
            msVal, _ := toString(m[msKey])
            if inVal != "" {
                ingredients = append(ingredients, bson.M{"name": inVal, "measure": msVal})
            }
        }

        // prepare doc. Keep raw under `raw` as well to preserve all fields
        doc := bson.M{
            "idMeal": id,
            "strMeal": mustString(m["strMeal"]),
            "strCategory": mustString(m["strCategory"]),
            "strArea": mustString(m["strArea"]),
            "strInstructions": mustString(m["strInstructions"]),
            "strMealThumb": mustString(m["strMealThumb"]),
            "strYoutube": mustString(m["strYoutube"]),
            "ingredients": ingredients,
            "raw": m,
            "updatedAt": time.Now(),
        }

        // upsert by idMeal
        filter := bson.M{"idMeal": id}
        update := bson.M{"$set": doc}
        _, err := coll.UpdateOne(s.ctx, filter, update, options.Update().SetUpsert(true))
        if err != nil {
            if s.log != nil {
                s.log.Errorw("failed to upsert meal", "id", id, "error", err)
            }
            return err
        }
    }

    return nil
}

func (s *Store) GetMealByID(id string) (map[string]interface{}, error) {
    key := "meal:" + id
    // try redis first
    if s.rdb != nil {
        if b, err := s.rdb.Get(s.ctx, key).Bytes(); err == nil {
            var out map[string]interface{}
            if err := json.Unmarshal(b, &out); err == nil {
                return out, nil
            }
        }
    }

    coll := s.db.Collection("meals")
    var result map[string]interface{}
    if err := coll.FindOne(s.ctx, bson.M{"idMeal": id}).Decode(&result); err != nil {
        if err == mongo.ErrNoDocuments {
            return nil, nil
        }
        return nil, err
    }

    // cache in redis
    if s.rdb != nil {
        if bb, err := json.Marshal(result); err == nil {
            _ = s.rdb.Set(s.ctx, key, bb, 24*time.Hour).Err()
        }
    }

    return result, nil
}

func toString(v interface{}) (string, bool) {
    if v == nil {
        return "", false
    }
    if s, ok := v.(string); ok {
        return s, true
    }
    return fmt.Sprint(v), true
}

func mustString(v interface{}) string {
    s, _ := toString(v)
    return s
}
