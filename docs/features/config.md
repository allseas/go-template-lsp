# Config File Support
You can toggle server features project wide by creating a ```gomtpl.config.json``` file in the root directory of your project.
An example file is given below
```json
{
  "enableServer": true,
  "trace": {
    "server": "messages" 
  }
}
```
A restart is needed for changes to apply.