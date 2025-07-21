package telegram

const (
	internalErrMsg           string = "что-то пошло не так..."
	booksNotFound            string = "не удалось найти книг..."
	requestTooOld            string = "время обработки запроса истекло, введите новый:"
	startBookDownloading     string = "начинаем скачивать книгу..."
	linkEmailText            string = "Введите ваш send-to-kindle email address. Найти его можно в вашем аккаунте Amazon (content & devices -> preferences -> personal document settings -> Send-to-Kindle E-Mail Settings)"
	EmailLinkedSuccessfully  string = "email успешно привязан"
	EmailDeletedSuccessfully string = "email успешно удален"
	StartSendingToKindle     string = "Начинаем отправку книги (процесс может занять до минуты для больших файлов)"
	EmailNotLinked           string = "У вас нет привязанного email, вы можете установить email отправив команду /email боту"
	BookSendedToKindle       string = "Книга успешно отправлена на ваш kindle. Если kindle подключен к wifi - то через несколько минут книга должна отобразиться. Возможно придет письмо от Amazon на почту, с подтверждением скачивания книги на kindle. Нужно нажать \"verify request\".\n\nЕсли книга не приходит - удостоверьтесь, что вы привязали правильный email, а также добавили адрес booksender@kotfedot-projects.ru в белый список отправителей. Подробнее в команде /help"
)
