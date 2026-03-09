### Environment Variables
Required environment variables. For discord info, just enable dev mode on discord and it will show more info across the app.
1. `BOT_TOKEN` - Your bot token from [discord developer portal](https://discord.com/developers/applications)
2. `GUILD_ID` - Server ID from the server you want to run the bot on
3. `DB_PATH` - SQLite db path, in this case its on a docker volume at `/data/docbot.db`
4. `WEB_BASE_URL` - Base URL for the server and used for sending links to view the read only info on the docs and reviewers info, i.e http://localhost:8080
5. `DISCORD_CHANNEL_ID` - Channel ID the bot will be using to send information (notification). In our case the ID for `#documentation`
6. `MAINTAINER_DISCORD_ID` - This is the user id for the maintainer of the docs (FYI gets lots of ping about everything going on). Right now we only have one hard coded maintainer. We can have as many reviewers as you want. We could probably work on this to add support for more maintainers in the future

### Deployment
The bot runs on docker and you can start it with `docker compose up --build`. We have a SQLite DB volume so this may need to be backed up. The cron job runs everyday Mon-Fri 9AM UTC+1
