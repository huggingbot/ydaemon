package controllers

import (
	"context"
	"net/http"
	"strconv"
	"strings"

	"github.com/ethereum/go-ethereum/common"
	"github.com/gin-gonic/gin"
	"github.com/machinebox/graphql"
	"github.com/majorfi/ydaemon/internal/ethereum"
	"github.com/majorfi/ydaemon/internal/logs"
	"github.com/majorfi/ydaemon/internal/models"
	"github.com/majorfi/ydaemon/internal/store"
	"github.com/majorfi/ydaemon/internal/utils"
)

func graphQLRequestForOneVault(vaultAddress string) *graphql.Request {
	return graphql.NewRequest(`{
		vault(id: "` + strings.ToLower(vaultAddress) + `") {
			id
			activation
			apiVersion
			classification
			managementFeeBps
			performanceFeeBps
			balanceTokens
			latestUpdate {
				timestamp
			}
			shareToken {
				name
				symbol
				id
				decimals
			}
			token {
				name
				symbol
				id
				decimals
			}
			strategies(first: 40) {
				address
				name
				inQueue
				debtLimit
			}
		}
	}`)
}

//GetAllVaults will, for a given chainID, return a list of all vaults
func (y controller) GetVault(c *gin.Context) {
	chainID, err := strconv.ParseUint(c.Param("chainID"), 10, 64)
	if err != nil {
		c.String(http.StatusBadRequest, "invalid chainID")
		return
	}
	vaultAddress := c.Param("address")
	if vaultAddress == `` {
		c.String(http.StatusBadRequest, "invalid address")
		return
	}

	if utils.ContainsAddress(utils.BLACKLISTED_VAULTS[chainID], common.HexToAddress(vaultAddress)) {
		c.String(http.StatusBadRequest, "invalid address")
		return
	}

	client := graphql.NewClient(ethereum.GetGraphURI(chainID))
	request := graphQLRequestForOneVault(vaultAddress)
	var response models.TGraphQueryResponseForVault
	if err := client.Run(context.Background(), request, &response); err != nil {
		logs.Error(err)
		c.String(http.StatusInternalServerError, "Impossible to fetch subgraph")
		return
	}

	strategiesCondition := selectStrategiesCondition(c.Query("strategiesCondition"))
	vaultFromGraph := response.Vault
	vaultFromMeta := store.VaultsFromMeta[chainID][common.HexToAddress(vaultFromGraph.Id).String()]
	shareTokenFromMeta := store.TokensFromMeta[chainID][common.HexToAddress(vaultFromGraph.ShareToken.Id).String()]
	tokenFromMeta := store.TokensFromMeta[chainID][common.HexToAddress(vaultFromGraph.Token.Id).String()]
	apyFromAPIV1 := store.VaultsFromAPIV1[chainID][common.HexToAddress(vaultFromGraph.Id).String()]
	strategiesFromMeta := store.StrategiesFromMeta[chainID]
	pricesForChainID := store.TokenPrices[chainID]

	c.JSON(http.StatusOK, prepareVaultSchema(
		chainID,
		strategiesCondition,
		vaultFromGraph,
		vaultFromMeta,
		shareTokenFromMeta,
		tokenFromMeta,
		strategiesFromMeta,
		apyFromAPIV1,
		pricesForChainID,
	))
}