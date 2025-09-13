> Guide 
> https://medium.com/data-science-collective/one-command-to-turn-any-csv-into-a-streamlit-app-with-gemini-cli-5ce43804a9cf

## Prompt with
```
Go with @data/<any>.csv

....

Add new tab for prediction

....

```

finally what we have <video controls src="media/demo.mp4" title="Title"></video>

## Install
```sh
uv sync
```

## Run with

```sh
uv run streamlit run app.py
```

## Using Makefile

To simplify development, a `Makefile` has been provided with the following commands:

- `make init`: Initializes the virtual environment and installs all necessary dependencies.
- `make run`: Starts the Streamlit application.
- `make clean`: Removes the virtual environment.
