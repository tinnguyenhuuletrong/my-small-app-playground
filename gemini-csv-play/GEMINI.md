# Project overview

You are helping to automate a data analysis workflow using the Gemini CLI.
You should:

1. Accept a CSV file. Unless otherwise stated, this will be in the folder 
'data'.
2. Propose a plan to analyse the given data.
3. The proposed plan should be presented to the user and can either be accepted as is or with modifications.
4. Generate a Streamlit app that implements the plan.
5. Use the 'uv' package manager to create and manage a virtual environment.
6. Automatically run the generated Streamlit app after setup.
7. Accept additional instructions to modify the running app.

# Workflow details

- Virtual environment name: '.venv' created using 'uv init'.
- Install dependencies: 'pandas', 'streamlit', 'Plotly' and any other libraries that are necessary using 'uv pip install <library>'
- Use Python 3.13 unless specified otherwise.
- Write all code into 'app.py' in the current directory.
- All charts are to be implemented using Plotly.
- The app should be run with the command: 'start uv run streamlit run app.py' (this will start a subprocess in Windows that will allow modification to run immediately)
- All tables should be displayed using Streamlit 'st.dataframe'.
- Use Streamlit user interface components to seperate the data views and navigation.

# Style preferences

- Code must be clean and modular. Use PEP8 formatting.
- Keep imports grouped at the top.
- Add clear inline comments explaining each major step.