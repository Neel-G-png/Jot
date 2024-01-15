# Jot - Simplifying Email Management and Note-Taking

Jot is a command-line tool designed to simplify your email management and note-taking process. It effortlessly reads and summarizes your Gmail content, then seamlessly integrates these summaries into your Notion workspace.

## Pre-Requisites

### 1. Setup Google Cloud Project and Enable Gmail API

- Navigate to [Google Cloud Console](https://console.cloud.google.com/projectcreate?authuser=1) and sign in with your Google account.
- Enter Project name and organization and click on **Create**.
- Click on **APIs and Services** -> **Enabled APIs and Services** in the Pinned Products on Dashboard view.
- Click on the **+ Enable APIS and SERVICES** button.
- Find the **Google Workspace - Gmail API** and click on the **Enable API** button.

### 2. Configure Oauth Consent

- In the Google Cloud console, go to **Menu** -> **APIs & Services** -> **OAuth consent screen**.
- Select the user type for your app and click **Create**.
- Complete the app registration form, then click **Save and Continue**.
- If you're creating an app for use outside of your Google Workspace organization, click **Add or Remove Scopes**. We recommend the following best practices when selecting scopes.
- After selecting the scopes required by your app, click **Save and Continue**.
- If you selected External for user type, add test users:
    - Under Test users, click **Add users**.
    - Enter your email address and any other authorized test users, then click **Save and Continue**.

### 3. Create Access Credentials

- In the Google Cloud console, go to **Menu** -> **APIs & Services** -> **Credentials**.
- Click **Create Credentials** > **OAuth client ID**.
- Click **Application type** > **Desktop app**.
- In the Name field, type a name for the credential. This name is only shown in the Google Cloud console.
- Click **Create**. The OAuth client created screen appears, showing your new Client ID and Client secret.
- Click **OK**. The newly created credential appears under OAuth 2.0 Client IDs.
- Download the JSON for the OAuth credentials and copy it to the Jot directory with the filename 'credentials.json'.

