package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"os"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
)

type binanceResp struct {
	Price float64 `json:"price,string"`
	Code  int64   `json:"code"`
}
type wallet map[string]float64

var db = map[int64]wallet{}

func main() {
	bot, err := tgbotapi.NewBotAPI(os.Getenv("TELEGRAM_TOKEN"))
	if err != nil {
		log.Panic(err)
	}

	log.Printf("Authorized on account %s", bot.Self.UserName)

	u := tgbotapi.NewUpdate(0)
	u.Timeout = 60

	updates := bot.GetUpdatesChan(u)

	for update := range updates {
		if update.Message == nil { // ignore any non-Message Updates
			continue
		}
		msgArr := strings.Split(update.Message.Text, " ")
		switch msgArr[0] {
		case "ADD":
			summ, err := strconv.ParseFloat(msgArr[2], 64)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Невозможно конвертировать валюту"))
				continue
			}
			if summ <= 0 {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Некорректная сумма"))
				continue
			}
			if _, ok := db[update.Message.Chat.ID]; !ok {
				db[update.Message.Chat.ID] = wallet{}
			}
			db[update.Message.Chat.ID][msgArr[1]] += summ
			msg := fmt.Sprintf("Баланс: %s %f", msgArr[1], db[update.Message.Chat.ID][msgArr[1]])
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))
		case "SUB":
			summ, err := strconv.ParseFloat(msgArr[2], 64)
			if err != nil {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Невозможно конвертироваать валюту"))
				continue
			}
			if summ <= 0 {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Некорректная сумма"))
				continue
			}
			if _, ok := db[update.Message.Chat.ID]; !ok {
				db[update.Message.Chat.ID] = wallet{}
			}
			if db[update.Message.Chat.ID][msgArr[1]] < summ {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Недостаточно средств"))
				continue
			}
			db[update.Message.Chat.ID][msgArr[1]] -= summ
			msg := fmt.Sprintf("Баланс: %s %f", msgArr[1], db[update.Message.Chat.ID][msgArr[1]])
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))
		case "DEL":
			delete(db[update.Message.Chat.ID], msgArr[1])
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Валюта Удалена"))
		case "SHOW":
			msg := "Баланс: \n"
			var UsdSumm float64
			for key, value := range db[update.Message.Chat.ID] {
				coinPrice, err := getPrice(key)
				if err != nil {
					bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, err.Error()))
					continue
				}
				UsdSumm += value * coinPrice
				msg += fmt.Sprintf("Сумма: %s %.2f [%.2f]\n", key, value, value*coinPrice)
			}
			msg += fmt.Sprintf("Сумма: %.2f долларов\n", UsdSumm)

			coinPriceRu, err := getPriceRU()
			if err != nil {
				bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, err.Error()))
				continue
			}
			RuSumm := UsdSumm * coinPriceRu
			msg += fmt.Sprintf("Сумма: %.2f рублей\n", RuSumm)
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, msg))
		default:
			bot.Send(tgbotapi.NewMessage(update.Message.Chat.ID, "Неизвестная команда"))
		}

	}
}

func getPrice(coin string) (price float64, err error) {
	resp, err := http.Get(fmt.Sprintf("https://api.binance.com/api/v3/ticker/price?symbol=%sUSDT", coin))
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var jsonResp binanceResp
	err = json.NewDecoder(resp.Body).Decode(&jsonResp)
	if err != nil {
		return
	}
	if jsonResp.Code != 0 {
		err = errors.New("Некорректная валюта")
		return

	}
	price = jsonResp.Price
	return
}

func getPriceRU() (price float64, err error) {
	resp, err := http.Get(fmt.Sprintf("https://api.binance.com/api/v3/ticker/price?symbol=USDTRUB"))
	if err != nil {
		return
	}
	defer resp.Body.Close()

	var jsonResp binanceResp
	err = json.NewDecoder(resp.Body).Decode(&jsonResp)
	if err != nil {
		return
	}
	if jsonResp.Code != 0 {
		err = errors.New("Некорректная валюта")
		return

	}
	price = jsonResp.Price
	return
}
