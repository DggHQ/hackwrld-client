package main

import (
	"github.com/prometheus/client_golang/prometheus"
)

type Monitor struct {
	MinedCoins  *prometheus.CounterVec
	SpentCoins  *prometheus.CounterVec
	LostCoins   *prometheus.CounterVec
	StolenCoins *prometheus.CounterVec
	CoolDown    *prometheus.GaugeVec
}

func (m *Monitor) Init() *Monitor {
	m.MinedCoins = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "hackwrld_mined_coins",
		Help: "Total amount of mined coins"}, []string{"id", "nick"})
	m.SpentCoins = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "hackwrld_spent_coins",
		Help: "Total amount of spent coins"}, []string{"id", "nick"})
	m.LostCoins = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "hackwrld_lost_coins",
		Help: "Total amount of coins lost to stealing"}, []string{"id", "nick", "stealerId", "stealerNick"})
	m.StolenCoins = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "hackwrld_stolen_coins",
		Help: "Total amount of coins gained by stealing"}, []string{"id", "nick"})
	m.CoolDown = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Name: "hackwrld_defender_cooldown",
		Help: "The current cooldown time of the player"}, []string{"id", "nick"})

	prometheus.MustRegister(m.MinedCoins)
	prometheus.MustRegister(m.SpentCoins)
	prometheus.MustRegister(m.LostCoins)
	prometheus.MustRegister(m.StolenCoins)
	prometheus.MustRegister(m.CoolDown)
	return m
}
