package ras

/*
	Function to send a update request to TSN-service, for the configuration.

	TODO: Currently, it only sends test data, it should be replaced with more relevant.
	TODO: Expand to update configuration for all relevant tables
*/

/* Update the configuration on the switches, so they all have an core configuration
 */
func UpdateDefaultConfig() (err error) {

	err = setConfigMstpPortTable(5, 250000, 1, 1, 1, true, false)
	if err != nil {
		return err
	}

	err = setConfigMstpCistPortTable(5, true, true, true, true, true, true, true, []byte("Hello world"), true, 0, 0, "", false, true)
	if err != nil {
		return err
	}

	return nil
}
