package i18n

import "github.com/karavanix/karavantrack-api-server/internal/domain/shared"

// Push notification keys. The ones about a load take ParamTitle.
const (
	// To the driver.
	PushLoadAssigned  Key = "push.load.assigned"
	PushLoadCancelled Key = "push.load.cancelled"
	PushLoadConfirmed Key = "push.load.confirmed"
	PushGPSDriver     Key = "push.gps.driver"

	// To the shipper side (load creator, company owners and admins).
	PushLoadAccepted       Key = "push.load.accepted"
	PushLoadPickupBegin    Key = "push.load.pickup_begin"
	PushLoadPickupConfirm  Key = "push.load.pickup_confirm"
	PushLoadStart          Key = "push.load.start"
	PushLoadDropoffBegin   Key = "push.load.dropoff_begin"
	PushLoadDropoffConfirm Key = "push.load.dropoff_confirm"
	PushGPSShipper         Key = "push.gps.shipper"
)

type pushText struct {
	title string
	body  string
}

// Uzbek is Latin script; oʻ/gʻ use U+02BB and the tutuq belgisi is U+02BC,
// as in the landing's glossary. "Shipper" is Joʻnatuvchi, the word the
// driver's app uses.
var pushes = map[Key]map[shared.Language]pushText{
	PushLoadAssigned: {
		shared.LanguageRussian: {"Новый груз", "Вам назначен новый груз: {{.title}}"},
		shared.LanguageUzbek:   {"Yangi yuk", "Sizga yangi yuk tayinlandi: {{.title}}"},
		shared.LanguageEnglish: {"New load", "You've been assigned a new load: {{.title}}"},
	},
	PushLoadCancelled: {
		shared.LanguageRussian: {"Груз отменён", "Владелец отменил груз: {{.title}}"},
		shared.LanguageUzbek:   {"Yuk bekor qilindi", "Joʻnatuvchi yukni bekor qildi: {{.title}}"},
		shared.LanguageEnglish: {"Load cancelled", "The shipper cancelled the load: {{.title}}"},
	},
	PushLoadConfirmed: {
		shared.LanguageRussian: {"Груз подтверждён", "Владелец подтвердил доставку: {{.title}}"},
		shared.LanguageUzbek:   {"Yuk tasdiqlandi", "Joʻnatuvchi yetkazishni tasdiqladi: {{.title}}"},
		shared.LanguageEnglish: {"Load confirmed", "The shipper confirmed delivery: {{.title}}"},
	},
	PushLoadAccepted: {
		shared.LanguageRussian: {"Груз принят", "Водитель принял груз: {{.title}}"},
		shared.LanguageUzbek:   {"Yuk qabul qilindi", "Haydovchi yukni qabul qildi: {{.title}}"},
		shared.LanguageEnglish: {"Load accepted", "The driver accepted the load: {{.title}}"},
	},
	PushLoadPickupBegin: {
		shared.LanguageRussian: {"Начало погрузки", "Водитель выезжает на погрузку: {{.title}}"},
		shared.LanguageUzbek:   {"Yuklash boshlandi", "Haydovchi yuklashga yoʻlga chiqdi: {{.title}}"},
		shared.LanguageEnglish: {"Heading to pickup", "The driver is on the way to pickup: {{.title}}"},
	},
	PushLoadPickupConfirm: {
		shared.LanguageRussian: {"Груз погружен", "Водитель подтвердил погрузку: {{.title}}"},
		shared.LanguageUzbek:   {"Yuk yuklandi", "Haydovchi yuklashni tasdiqladi: {{.title}}"},
		shared.LanguageEnglish: {"Cargo loaded", "The driver confirmed pickup: {{.title}}"},
	},
	PushLoadStart: {
		shared.LanguageRussian: {"Поездка начата", "Водитель начал поездку: {{.title}}"},
		shared.LanguageUzbek:   {"Reys boshlandi", "Haydovchi yoʻlga chiqdi: {{.title}}"},
		shared.LanguageEnglish: {"Trip started", "The driver started the trip: {{.title}}"},
	},
	PushLoadDropoffBegin: {
		shared.LanguageRussian: {"Прибытие на выгрузку", "Водитель прибыл на выгрузку: {{.title}}"},
		shared.LanguageUzbek:   {"Tushirishga yetib keldi", "Haydovchi tushirish joyiga yetib keldi: {{.title}}"},
		shared.LanguageEnglish: {"Arrived at dropoff", "The driver arrived at the dropoff: {{.title}}"},
	},
	PushLoadDropoffConfirm: {
		shared.LanguageRussian: {"Груз доставлен", "Водитель завершил доставку: {{.title}}"},
		shared.LanguageUzbek:   {"Yuk yetkazildi", "Haydovchi yetkazib berishni yakunladi: {{.title}}"},
		shared.LanguageEnglish: {"Cargo delivered", "The driver completed delivery: {{.title}}"},
	},
	PushGPSDriver: {
		shared.LanguageRussian: {"GPS не получен", "Нет обновления местоположения более 15 минут: {{.title}}"},
		shared.LanguageUzbek:   {"GPS signali yoʻq", "15 daqiqadan beri joylashuv yangilanmadi: {{.title}}"},
		shared.LanguageEnglish: {"No GPS signal", "No location update for over 15 minutes: {{.title}}"},
	},
	PushGPSShipper: {
		shared.LanguageRussian: {"Пропала связь с водителем", "Трекинг груза «{{.title}}» не обновляется более 15 минут"},
		shared.LanguageUzbek:   {"Haydovchi bilan aloqa uzildi", "«{{.title}}» yuki kuzatuvi 15 daqiqadan beri yangilanmayapti"},
		shared.LanguageEnglish: {"Lost contact with the driver", "Tracking for “{{.title}}” hasn't updated in over 15 minutes"},
	},
}
