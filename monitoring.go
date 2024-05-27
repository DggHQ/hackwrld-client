package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Monitor struct {
	MinedCoins  *prometheus.CounterVec
	Requests    *prometheus.CounterVec
	SpentCoins  *prometheus.CounterVec
	LostCoins   *prometheus.CounterVec
	StolenCoins *prometheus.CounterVec
	CoolDown    *prometheus.GaugeVec
}

func (m *Monitor) Init() *Monitor {
	m.Requests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "hackwrld_endpoint_request_count",
		Help: "Gets the request count for http endpoints"}, []string{"id", "nick", "endpoint"})
	m.MinedCoins = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "hackwrld_mined_coins",
		Help: "Total amount of mined coins"}, []string{"id", "nick", "team"})
	m.SpentCoins = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "hackwrld_spent_coins",
		Help: "Total amount of spent coins"}, []string{"id", "nick", "team"})
	m.LostCoins = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "hackwrld_lost_coins",
		Help: "Total amount of coins lost to stealing"}, []string{"id", "nick", "stealerId", "stealerNick", "stealerTeam"})
	m.StolenCoins = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "hackwrld_stolen_coins",
		Help: "Total amount of coins gained by stealing"}, []string{"id", "nick", "team"})
	m.CoolDown = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "hackwrld_defender_cooldown",
		Help: "The current cooldown time of the player"}, []string{"id", "nick"})

	prometheus.MustRegister(m.MinedCoins)
	prometheus.MustRegister(m.Requests)
	prometheus.MustRegister(m.SpentCoins)
	prometheus.MustRegister(m.LostCoins)
	prometheus.MustRegister(m.StolenCoins)
	prometheus.MustRegister(m.CoolDown)
	return m
}
