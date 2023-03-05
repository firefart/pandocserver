# pandocserver

this is a simple webserver to interact with pandoc and generate PDFs from markdown.

It uses the eisvogel template to all conversions so take a look at their yml options here [https://github.com/Wandmalfarbe/pandoc-latex-template](https://github.com/Wandmalfarbe/pandoc-latex-template).

You can modify the template using the yml options at the top. If you have embedded resources like images and backgrounds in your markdown file you must also supply them with the correct path.

To run the server it's easiest by using the provided docker image as it comes with everything installed.

You should not pass untrusted user input to this webserver and always filter your input. The binary is run with the --sandbox option but there are still ways to escape from the normal workflow.
## Options

```text
Usage of ./pandocserver:
  -command-timeout duration
        the timeout for the conversion command. Can also be set through the PANDOC_COMMAND_TIMEOUT environment variable. (default 1m0s)
  -debug
        Enable DEBUG mode. Can also be set through the PANDOC_DEBUG environment variable.
  -graceful-timeout duration
        the duration for which the server gracefully wait for existing connections to finish - e.g. 15s or 1m. Can also be set through the PANDOC_GRACEFUL_TIMEOUT environment variable. (default 5s)
  -host string
        IP and Port to bind to. Can also be set through the PANDOC_HOST environment variable. (default ":8080")
  -pandoc-data-dir string
        The pandoc data dir containing the templates. Can also be set through the PANDOC_DATA_DIR environment variable. (default "/.pandoc")
  -pandoc-path string
        The path of the pandoc binary. Can also be set through the PANDOC_PATH environment variable. (default "/usr/local/bin/pandoc")
```

## Running

The dockerimage already contains everything you need to get started. The image is published on Dockerhub [https://hub.docker.com/r/firefart/pandocserver](https://hub.docker.com/r/firefart/pandocserver) and the github container registry [https://github.com/firefart/pandocserver/pkgs/container/pandocserver](https://github.com/firefart/pandocserver/pkgs/container/pandocserver).

It's also advised the run the image with the `--init` flag as there are subprocess spawned that might not get cleaned up correctly. If using docker compose you can set `init: true` in the definition.

```text
docker pull golang:latest
docker pull pandoc/extra:latest
docker build --tag pandocserver:dev .
docker run --init --rm -p 8000:8000 pandocserver:dev -host 0.0.0.0:8000
```

Docker compose setup

```yml
pandocserver:
  image: firefart/pandocserver:latest
  restart: unless-stopped
  init: true
```

## Requests

To convert a document send a POST request to the `/convert` endpoint. The request needs to be JSON with the appropiate Content-Type header and the following structure:

```json
{
  "input": "Base64 encoded markdown template",
  "resources": {
    "background.pdf": "base64 encoded file content",
    "test.jpg": "base64 encoded file content"
  }
}
```

The `resources` object is optional and can be omitted if no resources are needed.

The returned response is also a JSON object with two possible outcomes. If the status code is not 200 there was an error. In this case the detailed error is shown on the terminal and a generic error message is sent back to the client.

Error JSON Response:

```json
{
  "error": "error message"
}
```

Default Response when status is 200:

```json
{
  "content": "base64 encoded PDF"
}
```

If the status code is 200 you will get a base64 encoded pdf file in the `content` object. Just base64decode the content and save it as a pdf.

## Example

### Basic

Example request to send the basic example from [https://raw.githubusercontent.com/Wandmalfarbe/pandoc-latex-template/master/examples/basic-example/document.md](https://raw.githubusercontent.com/Wandmalfarbe/pandoc-latex-template/master/examples/basic-example/document.md):

<code>curl -H 'content-type: application/json' -X POST --data '{"input":"LS0tCnRpdGxlOiAiRXhhbXBsZSBQREYiCmF1dGhvcjogW0F1dGhvcl0KZGF0ZTogIjIwMTctMDItMjAiCnN1YmplY3Q6ICJNYXJrZG93biIKa2V5d29yZHM6IFtNYXJrZG93biwgRXhhbXBsZV0KbGFuZzogImVuIgouLi4KCiMgVmluYXF1ZSBzYW5ndWluZSBtZXR1ZW50aSBjdWlxdWFtIEFsY3lvbmUgZml4dXMKCiMjIEFlc2N1bGVhZSBkb211cyB2aW5jZW11ciBldCBWZW5lcmlzIGFkc3VldHVzIGxhcHN1bQoKTG9yZW0gbWFya2Rvd251bSBMZXRvaWEsIGV0IGFsaW9zOiBmaWd1cmFlIGZsZWN0ZW50ZW0gYW5uaXMgYWxpcXVpZCBQZW5lb3NxdWUgYWIKZXNzZSwgb2JzdGF0IGdyYXZpdGF0ZS4gT2JzY3VyYSBhdHF1ZSBjb25pdWdlLCBwZXIgZGUgY29uaXVueCwgc2liaSAqKm1lZGlhcwpjb21tZW50YXF1ZSB2aXJnaW5lKiogYW5pbWEgdGFtZW4gY29taXRlbXF1ZSBwZXRpcywgc2VkLiBJbiBBbXBoaW9uIHZlc3Ryb3MKaGFtb3MgaXJlIGFyY2VvciBtYW5kZXJlIHNwaWN1bGEsIGluIGxpY2V0IGFsaXF1YW5kby4KCmBgYGphdmEKcHVibGljIGNsYXNzIEV4YW1wbGUgaW1wbGVtZW50cyBMb3JlbUlwc3VtIHsKCXB1YmxpYyBzdGF0aWMgdm9pZCBtYWluKFN0cmluZ1tdIGFyZ3MpIHsKCQlpZihhcmdzLmxlbmd0aCA8IDIpIHsKCQkJU3lzdGVtLm91dC5wcmludGxuKCJMb3JlbSBpcHN1bSBkb2xvciBzaXQgYW1ldCIpOwoJCX0KCX0gLy8gT2JzY3VyYSBhdHF1ZSBjb25pdWdlLCBwZXIgZGUgY29uaXVueAp9CmBgYAoKUG9ycmlnaXR1ciBldCBQYWxsYXMgbnVwZXIgbG9uZ3VzcXVlIGNyYXRlcmUgaGFidWlzc2Ugc2VwdWxjcm8gcGVjdG9yZSBmZXJ0dXIuCkxhdWRhdCBpbGxlIGF1ZGl0aTsgdmVydGl0dXIgaXVyYSB0dW0gbmVwb3RpcyBjYXVzYTsgbW90dXMuIERpdmEgdmlydHVzISBBY3JvdGEKZGVzdHJ1aXRpcyB2b3MgaXViZXQgcXVvIGV0IGNsYXNzaXMgZXhjZXNzZXJlIFNjeXJ1bXZlIHNwaXJvIHN1Yml0dXNxdWUgbWVudGUKUGlyaXRob2kgYWJzdHVsaXQsIGxhcGlkZXMuCgojIyBMeWRpYSBjYWVsbyByZWNlbnRpIGhhZXJlYmF0IGxhY2VydW0gcmF0YWUgYXQKClRlIGNvbmNlcGl0IHBvbGxpY2UgZnVnaXQgdmlhcyBhbHVtbm8gKipvcmFzKiogcXVhbSBwb3Rlc3QKW3J1cnN1c10oaHR0cDovL2V4YW1wbGUuY29tI3J1cnN1cykgb3B0YXQuIE5vbiBldmFkZXJlIG9yYmVtIGVxdW9ydW0sIHNwYXRpaXMsCnZlbCBwZWRlIGludGVyIHNpLgoKMS4gRGUgbmVxdWUgaXVyYSBhcXVpcwoyLiBGcmFuZ2l0dXIgZ2F1ZGlhIG1paGkgZW8gdW1vciB0ZXJyYWUgcXVvcwozLiBSZWNlbnMgZGlmZnVkaXQgaWxsZSB0YW50dW0KClxiZWdpbntlcXVhdGlvbn1cbGFiZWx7ZXE6bmVpZ2hib3ItcHJvcGFiaWxpdHl9CiAgICBwX3tpan0odCkgPSBcZnJhY3tcZWxsX2oodCkgLSBcZWxsX2kodCl9e1xzdW1fe2sgXGluIE5faSh0KX1ee30gXGVsbF9rKHQpIC0gXGVsbF9pKHQpfQpcZW5ke2VxdWF0aW9ufQoKVGFtZW4gY29uZGV0dXJxdWUgc2F4YSBQYWxsb3JxdWUgbnVtIGV0IGZlcmFydW0gcHJvbWl0dGlzIGludmVuaSBsaWxpYSBpdXZlbmNhZQphZGVzc2VudCBhcmJvci4gRmxvcmVudGUgcGVycXVlIGF0IGNvbmRldHVycXVlIHNheGEgZXQgZmVyYXJ1bSBwcm9taXR0aXMgdGVuZGViYXQuIEFybW9zIG5pc2kgb2JvcnRhcyByZWZ1Z2l0IG1lLgoKPiBFdCBuZXBvdGVzIHBvdGVyYXQsIHNlIHF1aS4gRXVudGVtIGVnbyBwYXRlciBkZXN1ZXRhcXVlIGFldGhlcmEgTWFlYW5kcmksIGV0CltEYXJkYW5pbyBnZW1pbmFxdWVdKGh0dHA6Ly9leGFtcGxlLmNvbSNEYXJkYW5pb19nZW1pbmFxdWUpIGNlcm5pdC4gTGFzc2FxdWUgcG9lbmFzCm5lYywgbWFuaWZlc3RhICRccGkgcl4yJCBtaXJhbnRpYSBjYXB0aXZhcnVtIHByb2hpYmViYW50IHNjZWxlcmF0byBncmFkdXMgdW51c3F1ZQpkdXJhLgoKLSBQZXJtdWxjZW5zIGZsZWJpbGUgc2ltdWwKLSBJdXJhIHR1bSBuZXBvdGlzIGNhdXNhIG1vdHVzIGRpdmEgdmlydHVzIEFjcm90YS4gVGFtZW4gY29uZGV0dXJxdWUgc2F4YSBQYWxsb3JxdWUgbnVtIGV0IGZlcmFydW0gcHJvbWl0dGlzIGludmVuaSBsaWxpYSBpdXZlbmNhZSBhZGVzc2VudCBhcmJvci4gRmxvcmVudGUgcGVycXVlIGF0IGlyZSBhcmN1bS4="}' http://localhost:8000/convert</code>

| A basic example page |
| :------------------: |
| ![A basic example page](/images/screen.png) |

### Custom Background

Custom Background using the following template [https://raw.githubusercontent.com/Wandmalfarbe/pandoc-latex-template/master/examples/title-page-background/document.md](https://raw.githubusercontent.com/Wandmalfarbe/pandoc-latex-template/master/examples/title-page-background/document.md). You need to supply the background in the ressources object. You can find some sample backgrounds over here [https://github.com/Wandmalfarbe/pandoc-latex-template/tree/master/examples/title-page-background/backgrounds](https://github.com/Wandmalfarbe/pandoc-latex-template/tree/master/examples/title-page-background/backgrounds).

This uses the following metadata:

```yml
titlepage: true,
titlepage-rule-color: "360049"
titlepage-background: "background1.pdf"
```

<code>curl -H 'content-type: application/json' -X POST --data '{"input":"LS0tCnRpdGxlOiAiVmluYXF1ZSBzYW5ndWluZSBtZXR1ZW50aSBjdWlxdWFtIEFsY3lvbmUgZml4dXMiCmF1dGhvcjogW0F1dGhvciBOYW1lXQpkYXRlOiAiMjAxNy0wMi0yMCIKc3ViamVjdDogIk1hcmtkb3duIgprZXl3b3JkczogW01hcmtkb3duLCBFeGFtcGxlXQpzdWJ0aXRsZTogIkFlc2N1bGVhZSBkb211cyB2aW5jZW11ciBldCBWZW5lcmlzIGFkc3VldHVzIGxhcHN1bSIKbGFuZzogImVuIgp0aXRsZXBhZ2U6IHRydWUsCnRpdGxlcGFnZS1ydWxlLWNvbG9yOiAiMzYwMDQ5Igp0aXRsZXBhZ2UtYmFja2dyb3VuZDogImJhY2tncm91bmQxLnBkZiIKLi4uCgojIFZpbmFxdWUgc2FuZ3VpbmUgbWV0dWVudGkgY3VpcXVhbSBBbGN5b25lIGZpeHVzCgojIyBBZXNjdWxlYWUgZG9tdXMgdmluY2VtdXIgZXQgVmVuZXJpcyBhZHN1ZXR1cyBsYXBzdW0KCkxvcmVtIG1hcmtkb3dudW0gTGV0b2lhLCBldCBhbGlvczogZmlndXJhZSBmbGVjdGVudGVtIGFubmlzIGFsaXF1aWQgUGVuZW9zcXVlIGFiCmVzc2UsIG9ic3RhdCBncmF2aXRhdGUuIE9ic2N1cmEgYXRxdWUgY29uaXVnZSwgcGVyIGRlIGNvbml1bngsIHNpYmkgKiptZWRpYXMKY29tbWVudGFxdWUgdmlyZ2luZSoqIGFuaW1hIHRhbWVuIGNvbWl0ZW1xdWUgcGV0aXMsIHNlZC4gSW4gQW1waGlvbiB2ZXN0cm9zCmhhbW9zIGlyZSBhcmNlb3IgbWFuZGVyZSBzcGljdWxhLCBpbiBsaWNldCBhbGlxdWFuZG8uCgpgYGBqYXZhCnB1YmxpYyBjbGFzcyBFeGFtcGxlIGltcGxlbWVudHMgTG9yZW1JcHN1bSB7CglwdWJsaWMgc3RhdGljIHZvaWQgbWFpbihTdHJpbmdbXSBhcmdzKSB7CgkJaWYoYXJncy5sZW5ndGggPCAyKSB7CgkJCVN5c3RlbS5vdXQucHJpbnRsbigiTG9yZW0gaXBzdW0gZG9sb3Igc2l0IGFtZXQiKTsKCQl9Cgl9IC8vIE9ic2N1cmEgYXRxdWUgY29uaXVnZSwgcGVyIGRlIGNvbml1bngKfQpgYGAKClBvcnJpZ2l0dXIgZXQgUGFsbGFzIG51cGVyIGxvbmd1c3F1ZSBjcmF0ZXJlIGhhYnVpc3NlIHNlcHVsY3JvIHBlY3RvcmUgZmVydHVyLgpMYXVkYXQgaWxsZSBhdWRpdGk7IHZlcnRpdHVyIGl1cmEgdHVtIG5lcG90aXMgY2F1c2E7IG1vdHVzLiBEaXZhIHZpcnR1cyEgQWNyb3RhCmRlc3RydWl0aXMgdm9zIGl1YmV0IHF1byBldCBjbGFzc2lzIGV4Y2Vzc2VyZSBTY3lydW12ZSBzcGlybyBzdWJpdHVzcXVlIG1lbnRlClBpcml0aG9pIGFic3R1bGl0LCBsYXBpZGVzLgoKIyMgTHlkaWEgY2FlbG8gcmVjZW50aSBoYWVyZWJhdCBsYWNlcnVtIHJhdGFlIGF0CgpUZSBjb25jZXBpdCBwb2xsaWNlIGZ1Z2l0IHZpYXMgYWx1bW5vICoqb3JhcyoqIHF1YW0gcG90ZXN0CltydXJzdXNdKGh0dHA6Ly9leGFtcGxlLmNvbSNydXJzdXMpIG9wdGF0LiBOb24gZXZhZGVyZSBvcmJlbSBlcXVvcnVtLCBzcGF0aWlzLAp2ZWwgcGVkZSBpbnRlciBzaS4KCjEuIERlIG5lcXVlIGl1cmEgYXF1aXMKMi4gRnJhbmdpdHVyIGdhdWRpYSBtaWhpIGVvIHVtb3IgdGVycmFlIHF1b3MKMy4gUmVjZW5zIGRpZmZ1ZGl0IGlsbGUgdGFudHVtCgpcYmVnaW57ZXF1YXRpb259XGxhYmVse2VxOm5laWdoYm9yLXByb3BhYmlsaXR5fQogICAgcF97aWp9KHQpID0gXGZyYWN7XGVsbF9qKHQpIC0gXGVsbF9pKHQpfXtcc3VtX3trIFxpbiBOX2kodCl9Xnt9IFxlbGxfayh0KSAtIFxlbGxfaSh0KX0KXGVuZHtlcXVhdGlvbn0KClRhbWVuIGNvbmRldHVycXVlIHNheGEgUGFsbG9ycXVlIG51bSBldCBmZXJhcnVtIHByb21pdHRpcyBpbnZlbmkgbGlsaWEgaXV2ZW5jYWUKYWRlc3NlbnQgYXJib3IuIEZsb3JlbnRlIHBlcnF1ZSBhdCBjb25kZXR1cnF1ZSBzYXhhIGV0IGZlcmFydW0gcHJvbWl0dGlzIHRlbmRlYmF0LiBBcm1vcyBuaXNpIG9ib3J0YXMgcmVmdWdpdCBtZS4KCkV0IG5lcG90ZXMgcG90ZXJhdCwgc2UgcXVpLiBFdW50ZW0gZWdvIHBhdGVyIGRlc3VldGFxdWUgYWV0aGVyYSBNYWVhbmRyaSwgZXQKW0RhcmRhbmlvIGdlbWluYXF1ZV0oaHR0cDovL2V4YW1wbGUuY29tI0RhcmRhbmlvX2dlbWluYXF1ZSkgY2Vybml0LiBMYXNzYXF1ZSBwb2VuYXMKbmVjLCBtYW5pZmVzdGEgJFxwaSByXjIkIG1pcmFudGlhIGNhcHRpdmFydW0gcHJvaGliZWJhbnQgc2NlbGVyYXRvIGdyYWR1cyB1bnVzcXVlCmR1cmEuCgotIFBlcm11bGNlbnMgZmxlYmlsZSBzaW11bAotIEl1cmEgdHVtIG5lcG90aXMgY2F1c2EgbW90dXMgZGl2YSB2aXJ0dXMgQWNyb3RhLiBUYW1lbiBjb25kZXR1cnF1ZSBzYXhhIFBhbGxvcnF1ZSBudW0gZXQgZmVyYXJ1bSBwcm9taXR0aXMgaW52ZW5pIGxpbGlhIGl1dmVuY2FlIGFkZXNzZW50IGFyYm9yLiBGbG9yZW50ZSBwZXJxdWUgYXQgaXJlIGFyY3VtLgo=", "resources": { "background1.pdf": "JVBERi0xLjUKJbXtrvsKMyAwIG9iago8PCAvTGVuZ3RoIDQgMCBSCiAgIC9GaWx0ZXIgL0ZsYXRlRGVjb2RlCj4+CnN0cmVhbQp4nGWOQQ7CQAhF93MKLiACAwMcwyO4sO2iLqz3T5zRpKlp2JDP/+/zKgxjthmud4L5XQhCGSMaWBqKO1x+gsP2gKkQSk3hBoTuYpp9iWqSOiB7phESMTx3RcMxvcEKtF9XWMo5MToipFrtVsKqRhkDTiASmKrdpCzIXA+of+zR+32a2Zt9gRJN3AaQPbDLwKRInN16hJwrlnJOTOVWPrFtQhIKZW5kc3RyZWFtCmVuZG9iago0IDAgb2JqCiAgIDE2MgplbmRvYmoKMiAwIG9iago8PAogICAvRXh0R1N0YXRlIDw8CiAgICAgIC9hMCA8PCAvQ0EgMSAvY2EgMSA+PgogICA+Pgo+PgplbmRvYmoKNSAwIG9iago8PCAvVHlwZSAvUGFnZQogICAvUGFyZW50IDEgMCBSCiAgIC9NZWRpYUJveCBbIDAgMCA1OTUuMjc1NTc0IDg0MS44ODk3NzEgXQogICAvQ29udGVudHMgMyAwIFIKICAgL0dyb3VwIDw8CiAgICAgIC9UeXBlIC9Hcm91cAogICAgICAvUyAvVHJhbnNwYXJlbmN5CiAgICAgIC9JIHRydWUKICAgICAgL0NTIC9EZXZpY2VSR0IKICAgPj4KICAgL1Jlc291cmNlcyAyIDAgUgo+PgplbmRvYmoKMSAwIG9iago8PCAvVHlwZSAvUGFnZXMKICAgL0tpZHMgWyA1IDAgUiBdCiAgIC9Db3VudCAxCj4+CmVuZG9iago2IDAgb2JqCjw8IC9DcmVhdG9yIChjYWlybyAxLjE0LjggKGh0dHA6Ly9jYWlyb2dyYXBoaWNzLm9yZykpCiAgIC9Qcm9kdWNlciAoY2Fpcm8gMS4xNC44IChodHRwOi8vY2Fpcm9ncmFwaGljcy5vcmcpKQo+PgplbmRvYmoKNyAwIG9iago8PCAvVHlwZSAvQ2F0YWxvZwogICAvUGFnZXMgMSAwIFIKPj4KZW5kb2JqCnhyZWYKMCA4CjAwMDAwMDAwMDAgNjU1MzUgZiAKMDAwMDAwMDU3NiAwMDAwMCBuIAowMDAwMDAwMjc2IDAwMDAwIG4gCjAwMDAwMDAwMTUgMDAwMDAgbiAKMDAwMDAwMDI1NCAwMDAwMCBuIAowMDAwMDAwMzQ4IDAwMDAwIG4gCjAwMDAwMDA2NDEgMDAwMDAgbiAKMDAwMDAwMDc2OCAwMDAwMCBuIAp0cmFpbGVyCjw8IC9TaXplIDgKICAgL1Jvb3QgNyAwIFIKICAgL0luZm8gNiAwIFIKPj4Kc3RhcnR4cmVmCjgyMAolJUVPRgo=" }}' http://localhost:8000/convert</code>

| A custom title page  | A basic example page |
| :------------------: | :------------------: |
| ![A custom title page](/images/title.png) | ![A basic example page](/images/screen.png) |
