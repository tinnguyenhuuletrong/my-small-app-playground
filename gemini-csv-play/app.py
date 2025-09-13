
import streamlit as st
import pandas as pd
import plotly.express as px

# Set page configuration
st.set_page_config(layout="wide")

# Function to load the data
@st.cache_data
def load_data():
    """
    Loads the Titanic dataset from a CSV file.
    """
    data = pd.read_csv('data/titanic.csv')
    return data

# Load the data
df = load_data()

# Sidebar for navigation
st.sidebar.title("Navigation")
page = st.sidebar.radio("Go to", ["Data Overview", "Survival Analysis", "Passenger Demographics"])

# Main content based on navigation
if page == "Data Overview":
    # Display a title for the data overview section
    st.title("Data Overview")

    # Display the raw data in a dataframe
    st.header("Raw Data")
    st.dataframe(df)

    # Display descriptive statistics of the data
    st.header("Descriptive Statistics")
    st.write(df.describe())

elif page == "Survival Analysis":
    # Display a title for the survival analysis section
    st.title("Survival Analysis")

    # Create two columns for layout
    col1, col2 = st.columns(2)

    with col1:
        # Display a header for the overall survival rate
        st.header("Overall Survival Rate")
        
        # Calculate survival counts
        survival_counts = df['Survived'].value_counts().reset_index()
        survival_counts.columns = ['Survived', 'Count']
        survival_counts['Survived'] = survival_counts['Survived'].map({1: 'Survived', 0: 'Did not survive'})
        
        # Create and display a pie chart for the overall survival rate
        fig = px.pie(survival_counts, names='Survived', values='Count', title='Overall Survival Rate')
        st.plotly_chart(fig)

    with col2:
        # Display a header for the survival rate by passenger class
        st.header("Survival Rate by Passenger Class")
        
        # Create and display a bar chart for survival rate by passenger class
        fig = px.bar(df.groupby('Pclass')['Survived'].mean().reset_index(), x='Pclass', y='Survived', title='Survival Rate by Pclass')
        st.plotly_chart(fig)

    # Create two more columns for layout
    col3, col4 = st.columns(2)

    with col3:
        # Display a header for the survival rate by sex
        st.header("Survival Rate by Sex")
        
        # Create and display a bar chart for survival rate by sex
        fig = px.bar(df.groupby('Sex')['Survived'].mean().reset_index(), x='Sex', y='Survived', title='Survival Rate by Sex')
        st.plotly_chart(fig)

    with col4:
        # Display a header for the survival rate by embarkation port
        st.header("Survival Rate by Embarkation Port")
        
        # Create and display a bar chart for survival rate by embarkation port
        fig = px.bar(df.groupby('Embarked')['Survived'].mean().reset_index(), x='Embarked', y='Survived', title='Survival Rate by Embarked Port')
        st.plotly_chart(fig)

elif page == "Passenger Demographics":
    # Display a title for the passenger demographics section
    st.title("Passenger Demographics")

    # Create two columns for layout
    col1, col2 = st.columns(2)

    with col1:
        # Display a header for the age distribution
        st.header("Age Distribution")
        
        # Create and display a histogram for age distribution
        fig = px.histogram(df, x='Age', nbins=30, title='Age Distribution')
        st.plotly_chart(fig)

    with col2:
        # Display a header for the fare distribution
        st.header("Fare Distribution")
        
        # Create and display a histogram for fare distribution
        fig = px.histogram(df, x='Fare', nbins=50, title='Fare Distribution')
        st.plotly_chart(fig)

    # Display a header for the relationship between age, fare, and class
    st.header("Relationship between Age, Fare, and Class")
    
    # Create and display a scatter plot for age vs. fare, colored by class
    fig = px.scatter(df, x='Age', y='Fare', color='Pclass', title='Age vs. Fare by Pclass')
    st.plotly_chart(fig)
