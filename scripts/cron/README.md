# Andromeda

## Cron job scripts

Sample cron configuration:

```
5 */6 * * * /home/myuser/go/src/jaytaylor.com/andromeda/scripts/cron/download-godoc-packages.sh 2>&1 1>/dev/null
45 */12 * * * /home/myuser/go/src/jaytaylor.com/andromeda/scripts/cron/trends.now.sh 2>&1 1>/dev/null
*/5 * * * * /home/myuser/go/src/jaytaylor.com/andromeda/scripts/cron/github.sh 2>&1 1>/dev/null
```
