package server

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	restful "github.com/emicklei/go-restful"

	"github.com/linkernetworks/mongo"
	"github.com/linkernetworks/vortex/src/config"
	"github.com/linkernetworks/vortex/src/entity"
	"github.com/linkernetworks/vortex/src/serviceprovider"
	"github.com/moby/moby/pkg/namesgenerator"
	"github.com/stretchr/testify/suite"

	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

type NetworkTestSuite struct {
	suite.Suite
	wc      *restful.Container
	session *mongo.Session
}

func (suite *NetworkTestSuite) SetupSuite() {
	cf := config.MustRead("../../config/testing.json")
	sp := serviceprovider.NewForTesting(cf)

	//init session
	suite.session = sp.Mongo.NewSession()
	//init restful container
	suite.wc = restful.NewContainer()
	service := newNetworkService(sp)
	suite.wc.Add(service)
}

func (suite *NetworkTestSuite) TearDownSuite() {
}

func TestNetworkSuite(t *testing.T) {
	suite.Run(t, new(NetworkTestSuite))
}

func (suite *NetworkTestSuite) TestCreateNetwork() {
	tName := namesgenerator.GetRandomName(0)
	network := entity.Network{
		Name: tName,
		Fake: entity.FakeNetwork{
			FakeParameter: "fake",
		},
		Type: "fake",
	}

	bodyBytes, err := json.MarshalIndent(network, "", "  ")
	suite.NoError(err)

	bodyReader := strings.NewReader(string(bodyBytes))
	httpRequest, err := http.NewRequest("POST", "http://localhost:7890/v1/networks", bodyReader)
	suite.NoError(err)

	httpRequest.Header.Add("Content-Type", "application/json")
	httpWriter := httptest.NewRecorder()
	suite.wc.Dispatch(httpWriter, httpRequest)
	//assertResponseCode(t, http.StatusOK, httpWriter)
	assertResponseCode(suite.T(), http.StatusOK, httpWriter)
	defer suite.session.Remove(entity.NetworkCollectionName, "name", tName)

	//We use the new write but empty input
	httpWriter = httptest.NewRecorder()
	suite.wc.Dispatch(httpWriter, httpRequest)
	assertResponseCode(suite.T(), http.StatusBadRequest, httpWriter)
	//Create again and it should fail since the name exist
	bodyReader = strings.NewReader(string(bodyBytes))
	httpRequest, err = http.NewRequest("POST", "http://localhost:7890/v1/networks", bodyReader)
	suite.NoError(err)
	httpRequest.Header.Add("Content-Type", "application/json")
	httpWriter = httptest.NewRecorder()
	suite.wc.Dispatch(httpWriter, httpRequest)
	assertResponseCode(suite.T(), http.StatusConflict, httpWriter)
}

func (suite *NetworkTestSuite) TestCreateWithInvalidParameter() {
	tName := namesgenerator.GetRandomName(0)
	network := entity.Network{
		Name: tName,
		Fake: entity.FakeNetwork{
			FakeParameter: "",
		},
		Type: "fake",
	}

	bodyBytes, err := json.MarshalIndent(network, "", "  ")
	suite.NoError(err)

	bodyReader := strings.NewReader(string(bodyBytes))
	httpRequest, err := http.NewRequest("POST", "http://localhost:7890/v1/networks", bodyReader)
	suite.NoError(err)

	httpRequest.Header.Add("Content-Type", "application/json")
	httpWriter := httptest.NewRecorder()
	suite.wc.Dispatch(httpWriter, httpRequest)
	assertResponseCode(suite.T(), http.StatusBadRequest, httpWriter)
}

func (suite *NetworkTestSuite) TestCreateWithCreteFail() {
	tName := namesgenerator.GetRandomName(0)
	network := entity.Network{
		Name: tName,
		Fake: entity.FakeNetwork{
			FakeParameter: "Yo",
			IWantFail:     true,
		},
		Type: "fake",
	}

	bodyBytes, err := json.MarshalIndent(network, "", "  ")
	suite.NoError(err)

	bodyReader := strings.NewReader(string(bodyBytes))
	httpRequest, err := http.NewRequest("POST", "http://localhost:7890/v1/networks", bodyReader)
	suite.NoError(err)

	httpRequest.Header.Add("Content-Type", "application/json")
	httpWriter := httptest.NewRecorder()
	suite.wc.Dispatch(httpWriter, httpRequest)
	assertResponseCode(suite.T(), http.StatusInternalServerError, httpWriter)
}

func (suite *NetworkTestSuite) TestDeleteNetwork() {
	tName := namesgenerator.GetRandomName(0)
	network := entity.Network{
		ID:   bson.NewObjectId(),
		Name: tName,
		Fake: entity.FakeNetwork{},
		Type: "fake",
	}

	//Create data into mongo manually
	suite.session.C(entity.NetworkCollectionName).Insert(network)
	defer suite.session.Remove(entity.NetworkCollectionName, "name", tName)

	httpRequestDelete, err := http.NewRequest("DELETE", "http://localhost:7890/v1/networks/"+network.ID.Hex(), nil)
	httpWriterDelete := httptest.NewRecorder()
	suite.wc.Dispatch(httpWriterDelete, httpRequestDelete)
	assertResponseCode(suite.T(), http.StatusOK, httpWriterDelete)
	err = suite.session.FindOne(entity.NetworkCollectionName, bson.M{"_id": network.ID}, &network)
	suite.Equal(err.Error(), mgo.ErrNotFound.Error())
}

func (suite *NetworkTestSuite) TestDeleteEmptyNetwork() {
	//Remove with non-exist network id
	httpRequest, err := http.NewRequest("DELETE", "http://localhost:7890/v1/networks/"+bson.NewObjectId().Hex(), nil)
	suite.NoError(err)

	httpWriter := httptest.NewRecorder()
	suite.wc.Dispatch(httpWriter, httpRequest)
	assertResponseCode(suite.T(), http.StatusNotFound, httpWriter)
}

//Fot Get/List, we only return mongo document
func (suite *NetworkTestSuite) TestGetNetwork() {
	tName := namesgenerator.GetRandomName(0)
	tType := "fake"

	network := entity.Network{
		ID:       bson.NewObjectId(),
		Name:     tName,
		Type:     tType,
		NodeName: namesgenerator.GetRandomName(0),
	}

	//Create data into mongo manually
	suite.session.C(entity.NetworkCollectionName).Insert(network)
	defer suite.session.Remove(entity.NetworkCollectionName, "name", tName)

	httpRequest, err := http.NewRequest("GET", "http://localhost:7890/v1/networks/"+network.ID.Hex(), nil)
	suite.NoError(err)

	httpWriter := httptest.NewRecorder()
	suite.wc.Dispatch(httpWriter, httpRequest)
	assertResponseCode(suite.T(), http.StatusOK, httpWriter)

	network = entity.Network{}
	err = json.Unmarshal(httpWriter.Body.Bytes(), &network)
	suite.NoError(err)
	suite.Equal(tName, network.Name)
	suite.Equal(tType, network.Type)
}

func (suite *NetworkTestSuite) TestGetNetworkWithInvalidID() {

	//Get data with non-exits ID
	httpRequest, err := http.NewRequest("GET", "http://localhost:7890/v1/networks/"+bson.NewObjectId().Hex(), nil)
	suite.NoError(err)

	httpWriter := httptest.NewRecorder()
	suite.wc.Dispatch(httpWriter, httpRequest)
	assertResponseCode(suite.T(), http.StatusNotFound, httpWriter)
}

func (suite *NetworkTestSuite) TestListNetwork() {
	networks := []entity.Network{}
	for i := 0; i < 3; i++ {
		networks = append(networks, entity.Network{
			Name:     namesgenerator.GetRandomName(0),
			Fake:     entity.FakeNetwork{},
			Type:     "fake",
			NodeName: namesgenerator.GetRandomName(0),
		})
	}

	for _, v := range networks {
		err := suite.session.C(entity.NetworkCollectionName).Insert(v)
		defer suite.session.Remove(entity.NetworkCollectionName, "name", v.Name)
		suite.NoError(err)
	}

	//list data by default page and page_size
	httpRequest, err := http.NewRequest("GET", "http://localhost:7890/v1/networks/", nil)
	suite.NoError(err)

	httpWriter := httptest.NewRecorder()
	suite.wc.Dispatch(httpWriter, httpRequest)
	assertResponseCode(suite.T(), http.StatusOK, httpWriter)

	retNetworks := []entity.Network{}
	err = json.Unmarshal(httpWriter.Body.Bytes(), &retNetworks)
	suite.NoError(err)
	suite.Equal(len(networks), len(retNetworks))
	fmt.Println(len(networks))
	fmt.Println(len(retNetworks))
	for i, v := range retNetworks {
		suite.Equal(networks[i].Name, v.Name)
		suite.Equal(networks[i].Type, v.Type)
		suite.Equal(networks[i].NodeName, v.NodeName)
	}

	//list data by different page and page_size
	httpRequest, err = http.NewRequest("GET", "http://localhost:7890/v1/networks?page=1&page_size=3", nil)
	suite.NoError(err)

	httpWriter = httptest.NewRecorder()
	suite.wc.Dispatch(httpWriter, httpRequest)
	assertResponseCode(suite.T(), http.StatusOK, httpWriter)

	retNetworks = []entity.Network{}
	err = json.Unmarshal(httpWriter.Body.Bytes(), &retNetworks)
	suite.NoError(err)
	suite.Equal(len(networks), len(retNetworks))
	for i, v := range retNetworks {
		suite.Equal(networks[i].Name, v.Name)
		suite.Equal(networks[i].Type, v.Type)
		suite.Equal(networks[i].NodeName, v.NodeName)
	}

	//list data by different page and page_size
	httpRequest, err = http.NewRequest("GET", "http://localhost:7890/v1/networks?page=1&page_size=1", nil)
	suite.NoError(err)

	httpWriter = httptest.NewRecorder()
	suite.wc.Dispatch(httpWriter, httpRequest)
	assertResponseCode(suite.T(), http.StatusOK, httpWriter)

	retNetworks = []entity.Network{}
	err = json.Unmarshal(httpWriter.Body.Bytes(), &retNetworks)
	suite.NoError(err)
	suite.Equal(1, len(retNetworks))
	for i, v := range retNetworks {
		suite.Equal(networks[i].Name, v.Name)
		suite.Equal(networks[i].Type, v.Type)
		suite.Equal(networks[i].NodeName, v.NodeName)
	}
}

func (suite *NetworkTestSuite) TestListNetworkWithInvalidPage() {
	//Get data with non-exits ID
	httpRequest, err := http.NewRequest("GET", "http://localhost:7890/v1/networks?page=asdd", nil)
	suite.NoError(err)

	httpWriter := httptest.NewRecorder()
	suite.wc.Dispatch(httpWriter, httpRequest)
	assertResponseCode(suite.T(), http.StatusBadRequest, httpWriter)

	httpRequest, err = http.NewRequest("GET", "http://localhost:7890/v1/networks?page_size=asdd", nil)
	suite.NoError(err)

	httpWriter = httptest.NewRecorder()
	suite.wc.Dispatch(httpWriter, httpRequest)
	assertResponseCode(suite.T(), http.StatusBadRequest, httpWriter)

	httpRequest, err = http.NewRequest("GET", "http://localhost:7890/v1/networks?page=-1", nil)
	suite.NoError(err)

	httpWriter = httptest.NewRecorder()
	suite.wc.Dispatch(httpWriter, httpRequest)
	assertResponseCode(suite.T(), http.StatusInternalServerError, httpWriter)
}
