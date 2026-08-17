# Microsoft Teams Presence Bot

Der Bot prüft seine Lizenz beim Start gegen den Lizenzserver.

Benötigte Umgebungsvariable:

- `LICENSE_KEY` – der im LicenseDeck erzeugte Lizenzschlüssel

Optional:

- `LICENSE_DEVICE_ID` – stabile ID der Installation; standardmäßig wird der Hostname verwendet
- `LICENSE_SERVER_URL` – abweichender Lizenzserver für Tests, standardmäßig `https://license.rindula.de`

Der Bot startet nicht, wenn die Lizenz ungültig ist, das Gerätelimit erreicht wurde oder der Lizenzserver nicht erreichbar ist. Die Lizenz wird während des Betriebs alle 15 Minuten erneut geprüft; bei einer späteren Ablehnung beendet sich der Bot ebenfalls. Die Prüfung sendet keine Microsoft- oder MQTT-Zugangsdaten an den Lizenzserver.
