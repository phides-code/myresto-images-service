package main

import (
	"context"
	"errors"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/expression"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/smithy-go"
	"github.com/google/uuid"
)

type Entity struct {
	Id        string `json:"id" dynamodbav:"id"`
	Content   string `json:"content" dynamodbav:"content"`
	CreatedOn uint64 `json:"createdOn" dynamodbav:"createdOn"`
}

type NewEntity struct {
	Content string `json:"content" validate:"required"`
}

type UpdatedEntity struct {
	Content string `json:"content" validate:"required"`
}

func getDbClient() (*dynamodb.Client, error) {
	sdkConfig, err := config.LoadDefaultConfig(context.TODO())
	dbClient := dynamodb.NewFromConfig(sdkConfig)

	return dbClient, err
}

func getEntity(ctx context.Context, id string) (*Entity, error) {
	key, err := attributevalue.Marshal(id)
	if err != nil {
		log.Println("getEntity() error running attributevalue.Marshal")
		return nil, err
	}

	input := &dynamodb.GetItemInput{
		TableName: aws.String(TableName),
		Key: map[string]types.AttributeValue{
			"id": key,
		},
	}

	result, err := db.GetItem(ctx, input)
	if err != nil {
		log.Println("getEntity() error running db.GetItem")
		return nil, err
	}

	if result.Item == nil {
		log.Println("getEntity() result.Item is nil")
		return nil, nil
	}

	entity := new(Entity)
	err = attributevalue.UnmarshalMap(result.Item, entity)
	if err != nil {
		log.Println("getEntity() error running attributevalue.UnmarshalMap")
		return nil, err
	}

	return entity, nil
}

func listEntities(ctx context.Context) ([]Entity, error) {
	entities := make([]Entity, 0)

	var token map[string]types.AttributeValue

	for {
		input := &dynamodb.ScanInput{
			TableName:         aws.String(TableName),
			ExclusiveStartKey: token,
		}

		result, err := db.Scan(ctx, input)
		if err != nil {
			log.Println("listEntities() error running db.Scan")
			return nil, err
		}

		var fetchedEntity []Entity
		err = attributevalue.UnmarshalListOfMaps(result.Items, &fetchedEntity)
		if err != nil {
			log.Println("listEntities() error running attributevalue.UnmarshalListOfMaps")
			return nil, err
		}

		entities = append(entities, fetchedEntity...)
		token = result.LastEvaluatedKey
		if token == nil {
			break
		}
	}

	return entities, nil
}

func insertEntity(ctx context.Context, newEntity NewEntity) (*Entity, error) {
	createdOnValue := uint64(time.Now().UnixMilli()) // Create a uint64 with current epoch

	entity := Entity{
		Id:        uuid.NewString(),
		Content:   newEntity.Content,
		CreatedOn: createdOnValue,
	}

	entityMap, err := attributevalue.MarshalMap(entity)
	if err != nil {
		log.Println("insertEntity() error running attributevalue.MarshalMap")
		return nil, err
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(TableName),
		Item:      entityMap,
	}

	res, err := db.PutItem(ctx, input)
	if err != nil {
		log.Println("insertEntity() error running db.PutItem")
		return nil, err
	}

	err = attributevalue.UnmarshalMap(res.Attributes, &entity)
	if err != nil {
		log.Println("insertEntity() error running attributevalue.UnmarshalMap")
		return nil, err
	}

	return &entity, nil
}

func updateEntity(ctx context.Context, id string, updatedEntity UpdatedEntity) (*Entity, error) {
	key, err := attributevalue.Marshal(id)
	if err != nil {
		log.Println("updateEntity() error running attributevalue.Marshal")
		return nil, err
	}

	expr, err := expression.NewBuilder().WithUpdate(
		expression.Set(
			expression.Name("content"),
			expression.Value(updatedEntity.Content),
		),
	).WithCondition(
		expression.Equal(
			expression.Name("id"),
			expression.Value(id),
		),
	).Build()
	if err != nil {
		log.Println("updateEntity error running expression.NewBuilder")
		return nil, err
	}

	input := &dynamodb.UpdateItemInput{
		Key: map[string]types.AttributeValue{
			"id": key,
		},
		TableName:                 aws.String(TableName),
		UpdateExpression:          expr.Update(),
		ExpressionAttributeNames:  expr.Names(),
		ExpressionAttributeValues: expr.Values(),

		ConditionExpression: expr.Condition(),
		ReturnValues:        types.ReturnValue(*aws.String("ALL_NEW")),
	}

	res, err := db.UpdateItem(ctx, input)
	if err != nil {
		var smErr *smithy.OperationError
		if errors.As(err, &smErr) {
			var condCheckFailed *types.ConditionalCheckFailedException
			if errors.As(err, &condCheckFailed) {
				log.Println("updateEntity() error running db.UpdateItem: Conditional check failed")
				return nil, nil
			}
		}

		log.Println("updateEntity() error running db.UpdateItem")
		return nil, err
	}

	if res.Attributes == nil {
		log.Println("updateEntity() error: res.Attributes == nil - Entity not found")
		return nil, nil
	}

	entity := new(Entity)
	err = attributevalue.UnmarshalMap(res.Attributes, entity)
	if err != nil {
		log.Println("updateEntity() error running attributevalue.UnmarshalMap")
		return nil, err
	}

	return entity, nil
}

func deleteEntity(ctx context.Context, id string) (*Entity, error) {
	key, err := attributevalue.Marshal(id)
	if err != nil {
		log.Println("deleteEntity() error running attributevalue.Marshal")
		return nil, err
	}

	input := &dynamodb.DeleteItemInput{
		TableName: aws.String(TableName),
		Key: map[string]types.AttributeValue{
			"id": key,
		},
		ReturnValues: types.ReturnValue(*aws.String("ALL_OLD")),
	}

	res, err := db.DeleteItem(ctx, input)
	if err != nil {
		log.Println("deleteEntity() error running db.DeleteItem")
		return nil, err
	}

	if res.Attributes == nil {
		log.Println("deleteEntity() error: res.Attributes == nil - Entity not found")
		return nil, nil
	}

	entity := new(Entity)
	err = attributevalue.UnmarshalMap(res.Attributes, entity)
	if err != nil {
		log.Println("deleteEntity() error running attributevalue.UnmarshalMap")
		return nil, err
	}

	return entity, nil
}
