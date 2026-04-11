package models

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/josephalai/sentanyl/pkg/models"
	"github.com/josephalai/sentanyl/pkg/utils"

	"gopkg.in/mgo.v2/bson"
)

type Creator struct {
	Id       bson.ObjectId `bson:"_id" json:"id,omitempty" required:"true"`
	PublicId string        `bson:"public_id" json:"public_id,omitempty" required:"true"`
	Email    string        `bson:"email" json:"email,omitempty" required:"true"`
	Name     struct {
		FirstName  string `bson:"first_name" json:"first_name,omitempty" required:"true"`
		MiddleName string `bson:"middle_name" json:"middle_name,omitempty"`
		LastName   string `bson:"last_name" json:"last_name,omitempty" required:"true"`
	} `bson:"name" json:"name,omitempty" diff:"name"`
	Password  string          `bson:"password" json:"password,omitempty" required:"true"`
	Stories   []*bson.ObjectId `bson:"stories,omitempty" json:"stories,omitempty"`
	Templates []*bson.ObjectId `bson:"templates,omitempty" json:"templates,omitempty"`
	Lists     []*EmailList    `bson:"lists,omitempty" json:"lists,omitempty"`
	ReplyTo   string          `bson:"reply_to" json:"reply_to,omitempty" required:"true"`

	models.Timestamps `bson:"timestamps,omitempty" json:"timestamps,omitempty"`
}

func NewCreator() *Creator {
	return &Creator{
		Id:       bson.NewObjectId(),
		PublicId: utils.GeneratePublicId(),
	}
}

func (c *Creator) GetIdHex() string {
	return fmt.Sprintf("%v", c.Id.Hex())
}

func (c *Creator) GetId() bson.ObjectId {
	return c.Id
}

func (c *Creator) ToJson() []byte {
	b, err := json.Marshal(c)
	if err != nil {
		log.Printf("error marshalling Creator: %v", err)
	}
	return b
}

func (c *Creator) ToJsonPretty() []byte {
	b, err := json.MarshalIndent(c, "", "    ")
	if err != nil {
		log.Printf("error marshalling Creator: %v", err)
	}
	return b
}

func (c *Creator) FromJson(jsonData []byte) {
	err := json.Unmarshal(jsonData, c)
	if err != nil {
		log.Printf("error unmarshalling Creator: %v", err)
	}
}

type EmailList struct {
	Id        bson.ObjectId   `bson:"_id" json:"id,omitempty"`
	CreatorId bson.ObjectId   `bson:"creator_id,omitempty" json:"creator_id,omitempty"`
	PublicId  string          `bson:"public_id" json:"public_id,omitempty"`
	Name      string          `bson:"name,omitempty" json:"name,omitempty"`
	Active    []bson.ObjectId `bson:"active,omitempty" json:"active,omitempty"`
	Removed   []bson.ObjectId `bson:"removed,omitempty" json:"removed,omitempty"`

	models.SoftDeletes `bson:"timestamps,omitempty" json:"timestamps,omitempty"`
}

func NewEmailList() *EmailList {
	return &EmailList{Id: bson.NewObjectId(), PublicId: utils.GeneratePublicId()}
}

func (e *EmailList) GetIdHex() string {
	return fmt.Sprintf("%v", e.Id.Hex())
}

func (e *EmailList) GetId() bson.ObjectId {
	return e.Id
}

func (e *EmailList) ToJson() []byte {
	b, err := json.Marshal(e)
	if err != nil {
		log.Printf("error marshalling EmailList: %v", err)
	}
	return b
}

// FindInSlice returns the smallest index i at which x == a[i],
// or len(a) if there is no such index.
func (e *EmailList) FindInSlice(a []bson.ObjectId, x bson.ObjectId) int {
	for i, n := range a {
		if x == n {
			return i
		}
	}
	return len(a)
}

// SliceContains tells whether a contains x.
func (e *EmailList) SliceContains(a []bson.ObjectId, x bson.ObjectId) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}
