- get data from session file, use diff
- after open the server, kill all unused tmux sessions

fuck just found this:
https://github.com/anthropics/claude-code/issues/1335

- add select num, to select and enter

- status: needPermission, stop, running

 write a skill.md to instruct agents how to use the tool. write in english. just use client. I will only provide the client exec in skill's folder. tell agent to use create, list to get basic context, and use get [limit] to get output for context. also, use info to get it's status. explain stop, running, needpermission. stop so you can get output and ask again. running to wait. needpermission to get

write a skill.md to instruct agents how to use the tool. write in english. just use client. I will only provide the client exec in skill's folder. tell agent to use create, list to get basic context, and use get [limit] to get output for context. also, use info to get it's status. explain stop, running, needpermission. stop so you can get output and ask again. running to wait. needpermission to get and see. if you should allow you send "Up", "Down", "Enter" to select (do it seperately). use input to send prompt. when you want to submitted the prompt use input "Enter" (send the prompt and "Enter" seperately). after the job. use delete to delete a session.
