## Original idea and philosophy

Python installer for windows that doesn't require an enterprise IT department or attempt to integrate with one.

An installer that isn't arrogant and elitist and require you to understand the internals of the installer and operating
system to install or to be a machine who doesn't need to understand.

## Feature wish list

### Usability

- "Next, next, next, finish" should be a correct option
- All default installation works or the installer refuses to run. For example, if other applications need to be
  uninstalled first, then the installer will refuse to run until the other applications are uninstalled.

### Removing Marketing Concerns of one Company

- No money grabs with commercial stores.
- No requirement for a virtual machine like WSL or a Macintosh. Some people are stuck with Windows for one reason or
  another and do not want to participate in the further enrichment of the elites at Microsoft.
- No enterprise concerns, such as Active Directory integration
- No "Add Remove/Programs", "Settings", "Control Panel"
- No registry usage, no registry corruption
- No usage of "Program Files"
- No cross promoting of Microsoft products and services
- No integration with Edge browser
- Default to platform dirs (AppData, etc) but provide options to not use it.
- No cmd.exe or Powershell dependencies, if possible
- No Trademarks or restricted intellectual property of Microsoft's.

### Installer Shape

- Single .exe binary
- Single .zip
- Option to create your own installer from source
- Some sort of upgrade path
- Possible Scoop or NSIS support
- PATH support baked in with a reasonable default
- Is not itself Python
- No C or C++
- Crossplatform as a check on if it has too many Windows-only concerns, but not expectation that this would be used
  anywhere else.

## AI Use

- Will be written by LLMs. Anti-LLM scolds are elitists that weren't going to participate anyhow & on average aren't
  Windows users and can afford a Macintosh.
- Before final, AI text should be updated and more importantly, the default experience should not require reading
  anything.
