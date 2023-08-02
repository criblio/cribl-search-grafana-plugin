# Grafana Cribl Plugin

Welcome to the Grafana Cribl plugin README. This plugin empowers you to create Grafana data sources pulling data directly from [Cribl Search](https://cribl.io/search/), with the resulting data visualizations offering insights like never before. Let's get into how to set this up.

## How It Works

1. **Set up a scheduled search in Cribl:** First, save a search in Cribl and configure it to run on a scheduled basis. Make a note of this search's ID. Learn how to set up a scheduled search with this [guide](https://docs.cribl.io/search/scheduled-searches/).
   
2. **Install this plugin to your Grafana deployment:** Add this plugin to your Grafana deployment to enable data source creation.

3. **Create a data source in Grafana using this plugin:** Configure this data source using your Cribl client ID and client secret. Details on how to acquire these can be found in the next section.

4. **Query the data source with your Cribl scheduled search ID:** The data source will fetch the latest results from the scheduled search based on the search ID you input.

That's the nutshell of the plugin's workings!

## Getting Your Cribl Client ID and Client Secret

To set up the data source in Grafana, you will need to input your Cribl client ID and client secret. Here's how you find these:

1. Navigate to the Cribl cloud and select your organization.

2. Look for the API management section.

3. Copy your client ID and client secret.

4. Back in Grafana, while adding a data source, paste the copied Cribl organization URL and the credentials.

5. Click on "Save and Test" to ensure everything is working correctly.

## Support & Feedback

We're always looking forward to hearing about your experience with the plugin. If you have any questions, suggestions, or feedback, don't hesitate to reach out:

- Join the conversation at the Cribl community: [https://cribl.io/community/](https://cribl.io/community/)
- Dig deeper with our documentation: [https://docs.cribl.io/](https://docs.cribl.io/)
- Community Slack cribl-community.slack.com
