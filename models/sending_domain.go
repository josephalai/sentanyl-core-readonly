package models

import (
	"encoding/json"
	"fmt"
	"log"

	"github.com/josephalai/sentanyl/pkg/models"
	"github.com/josephalai/sentanyl/pkg/utils"

	"gopkg.in/mgo.v2/bson"
)

type SendingDomain struct {
	Id         bson.ObjectId `bson:"_id" json:"id"`
	PublicId   string        `bson:"public_id" json:"public_id"`
	CreatorId  string        `bson:"creator_id" json:"creator_id"`
	Domain     string        `bson:"domain" json:"domain"`
	Selector   string        `bson:"selector" json:"selector"`
	VMTA       string        `bson:"vmta" json:"vmta"`
	PublicKey  string        `bson:"public_key" json:"public_key"`
	PrivateKey string        `bson:"private_key" json:"-"`
	Status     string        `bson:"status" json:"status"`
	DNSRecords DNSRecords    `bson:"dns_records" json:"dns_records"`

	models.SoftDeletes `bson:"timestamps,omitempty" json:"timestamps,omitempty"`
}

type DNSRecords struct {
	SPF      string `bson:"spf" json:"spf"`
	DKIM     string `bson:"dkim" json:"dkim"`
	DKIMName string `bson:"dkim_name" json:"dkim_name"`
	MX       string `bson:"mx" json:"mx"`
}

const (
	DomainStatusPendingDNS = "pending_dns"
	DomainStatusActive     = "active"
	DomainStatusPaused     = "paused"
)

func NewSendingDomain() *SendingDomain {
	return &SendingDomain{
		Id:       bson.NewObjectId(),
		PublicId: utils.GeneratePublicId(),
		Status:   DomainStatusPendingDNS,
	}
}

func (d *SendingDomain) GetIdHex() string {
	return fmt.Sprintf("%v", d.Id.Hex())
}

func (d *SendingDomain) GetId() bson.ObjectId {
	return d.Id
}

func (d *SendingDomain) ToJson() []byte {
	b, err := json.Marshal(d)
	if err != nil {
		log.Printf("error marshalling SendingDomain: %v", err)
	}
	return b
}
