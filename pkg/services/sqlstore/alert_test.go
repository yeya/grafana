// +build integration

package sqlstore

import (
	"testing"
	"time"

	"github.com/grafana/grafana/pkg/components/simplejson"
	"github.com/grafana/grafana/pkg/models"
)

func mockTimeNow() {
	var timeSeed int64
	timeNow = func() time.Time {
		loc := time.FixedZone("MockZoneUTC-5", -5*60*60)
		fakeNow := time.Unix(timeSeed, 0).In(loc)
		timeSeed++
		return fakeNow
	}
}

func resetTimeNow() {
	timeNow = time.Now
}

func TestAlertingDataAccess(t *testing.T) {
	mockTimeNow()
	defer resetTimeNow()

	t.Run("Testing Alerting data access", func(t *testing.T) {
		sqlStore := InitTestDB(t)

		testDash := insertTestDashboard(t, sqlStore, "dashboard with alerts", 1, 0, false, "alert")
		evalData, err := simplejson.NewJson([]byte(`{"test": "test"}`))
		require.NoError(t, err)
		items := []*models.Alert{
			{
				PanelId:     1,
				DashboardId: testDash.Id,
				OrgId:       testDash.OrgId,
				Name:        "Alerting title",
				Message:     "Alerting message",
				Settings:    simplejson.New(),
				Frequency:   1,
				EvalData:    evalData,
			},
		}

		cmd := models.SaveAlertsCommand{
			Alerts:      items,
			DashboardId: testDash.Id,
			OrgId:       1,
			UserId:      1,
		}

		err = SaveAlerts(&cmd)

		t.Run("Can create one alert", func(t *testing.T) {
			require.NoError(t, err)
		})

		t.Run("Can set new states", func(t *testing.T) {

			// Get alert so we can use its ID in tests
			alertQuery := models.GetAlertsQuery{DashboardIDs: []int64{testDash.Id}, PanelId: 1, OrgId: 1, User: &models.SignedInUser{OrgRole: models.ROLE_ADMIN}}
			err2 := HandleAlertsQuery(&alertQuery)
			require.Nil(t, err2)

			insertedAlert := alertQuery.Result[0]

			t.Run("new state ok", func(t *testing.T) {
				cmd := &models.SetAlertStateCommand{
					AlertId: insertedAlert.Id,
					State:   models.AlertStateOK,
				}

				err = SetAlertState(cmd)
				require.NoError(t, err)
			})

			alert, _ := getAlertById(insertedAlert.Id)
			stateDateBeforePause := alert.NewStateDate

			t.Run("can pause all alerts", func(t *testing.T) {
				err := pauseAllAlerts(true)
				require.NoError(t, err)

				t.Run("cannot updated paused alert", func(t *testing.T) {
					cmd := &models.SetAlertStateCommand{
						AlertId: insertedAlert.Id,
						State:   models.AlertStateOK,
					}

					err = SetAlertState(cmd)
					require.Error(t, err)
				})

				t.Run("alert is paused", func(t *testing.T) {
					alert, _ = getAlertById(insertedAlert.Id)
					currentState := alert.State
					require.Equal(t, "paused", currentState)
				})

				t.Run("pausing alerts should update their NewStateDate", func(t *testing.T) {
					alert, _ = getAlertById(insertedAlert.Id)
					stateDateAfterPause := alert.NewStateDate
					So(stateDateBeforePause, ShouldHappenBefore, stateDateAfterPause)
				})

				t.Run("unpausing alerts should update their NewStateDate again", func(t *testing.T) {
					err := pauseAllAlerts(false)
					require.NoError(t, err)
					alert, _ = getAlertById(insertedAlert.Id)
					stateDateAfterUnpause := alert.NewStateDate
					So(stateDateBeforePause, ShouldHappenBefore, stateDateAfterUnpause)
				})
			})
		})

		t.Run("Can read properties", func(t *testing.T) {
			alertQuery := models.GetAlertsQuery{DashboardIDs: []int64{testDash.Id}, PanelId: 1, OrgId: 1, User: &models.SignedInUser{OrgRole: models.ROLE_ADMIN}}
			err2 := HandleAlertsQuery(&alertQuery)

			alert := alertQuery.Result[0]
			require.Nil(t, err2)
			So(alert.Id, ShouldBeGreaterThan, 0)
			require.Equal(t, testDash.Id, alert.DashboardId)
			require.Equal(t, 1, alert.PanelId)
			require.Equal(t, "Alerting title", alert.Name)
			require.Equal(t, models.AlertStateUnknown, alert.State)
			require.NotNil(t, alert.NewStateDate)
			require.NotNil(t, alert.EvalData)
			require.Equal(t, "test", alert.EvalData.Get("test").MustString())
			require.NotNil(t, alert.EvalDate)
			require.Equal(t, "", alert.ExecutionError)
			require.NotNil(t, alert.DashboardUid)
			require.Equal(t, "dashboard-with-alerts", alert.DashboardSlug)
		})

		t.Run("Viewer can read alerts", func(t *testing.T) {
			viewerUser := &models.SignedInUser{OrgRole: models.ROLE_VIEWER, OrgId: 1}
			alertQuery := models.GetAlertsQuery{DashboardIDs: []int64{testDash.Id}, PanelId: 1, OrgId: 1, User: viewerUser}
			err2 := HandleAlertsQuery(&alertQuery)

			require.Nil(t, err2)
			require.Equal(t, 1, len(alertQuery.Result))
		})

		t.Run("Alerts with same dashboard id and panel id should update", func(t *testing.T) {
			modifiedItems := items
			modifiedItems[0].Name = "Name"

			modifiedCmd := models.SaveAlertsCommand{
				DashboardId: testDash.Id,
				OrgId:       1,
				UserId:      1,
				Alerts:      modifiedItems,
			}

			err := SaveAlerts(&modifiedCmd)

			t.Run("Can save alerts with same dashboard and panel id", func(t *testing.T) {
				require.NoError(t, err)
			})

			t.Run("Alerts should be updated", func(t *testing.T) {
				query := models.GetAlertsQuery{DashboardIDs: []int64{testDash.Id}, OrgId: 1, User: &models.SignedInUser{OrgRole: models.ROLE_ADMIN}}
				err2 := HandleAlertsQuery(&query)

				require.Nil(t, err2)
				require.Equal(t, 1, len(query.Result))
				require.Equal(t, "Name", query.Result[0].Name)

				t.Run("Alert state should not be updated", func(t *testing.T) {
					require.Equal(t, models.AlertStateUnknown, query.Result[0].State)
				})
			})

			t.Run("Updates without changes should be ignored", func(t *testing.T) {
				err3 := SaveAlerts(&modifiedCmd)
				require.Nil(t, err3)
			})
		})

		t.Run("Multiple alerts per dashboard", func(t *testing.T) {
			multipleItems := []*models.Alert{
				{
					DashboardId: testDash.Id,
					PanelId:     1,
					Name:        "1",
					OrgId:       1,
					Settings:    simplejson.New(),
				},
				{
					DashboardId: testDash.Id,
					PanelId:     2,
					Name:        "2",
					OrgId:       1,
					Settings:    simplejson.New(),
				},
				{
					DashboardId: testDash.Id,
					PanelId:     3,
					Name:        "3",
					OrgId:       1,
					Settings:    simplejson.New(),
				},
			}

			cmd.Alerts = multipleItems
			err = SaveAlerts(&cmd)

			t.Run("Should save 3 dashboards", func(t *testing.T) {
				require.NoError(t, err)

				queryForDashboard := models.GetAlertsQuery{DashboardIDs: []int64{testDash.Id}, OrgId: 1, User: &models.SignedInUser{OrgRole: models.ROLE_ADMIN}}
				err2 := HandleAlertsQuery(&queryForDashboard)

				require.Nil(t, err2)
				require.Equal(t, 3, len(queryForDashboard.Result))
			})

			t.Run("should updated two dashboards and delete one", func(t *testing.T) {
				missingOneAlert := multipleItems[:2]

				cmd.Alerts = missingOneAlert
				err = SaveAlerts(&cmd)

				t.Run("should delete the missing alert", func(t *testing.T) {
					query := models.GetAlertsQuery{DashboardIDs: []int64{testDash.Id}, OrgId: 1, User: &models.SignedInUser{OrgRole: models.ROLE_ADMIN}}
					err2 := HandleAlertsQuery(&query)
					require.Nil(t, err2)
					require.Equal(t, 2, len(query.Result))
				})
			})
		})

		t.Run("When dashboard is removed", func(t *testing.T) {
			items := []*models.Alert{
				{
					PanelId:     1,
					DashboardId: testDash.Id,
					Name:        "Alerting title",
					Message:     "Alerting message",
				},
			}

			cmd := models.SaveAlertsCommand{
				Alerts:      items,
				DashboardId: testDash.Id,
				OrgId:       1,
				UserId:      1,
			}

			err = SaveAlerts(&cmd)
			require.NoError(t, err)

			err = DeleteDashboard(&models.DeleteDashboardCommand{
				OrgId: 1,
				Id:    testDash.Id,
			})
			require.NoError(t, err)

			t.Run("Alerts should be removed", func(t *testing.T) {
				query := models.GetAlertsQuery{DashboardIDs: []int64{testDash.Id}, OrgId: 1, User: &models.SignedInUser{OrgRole: models.ROLE_ADMIN}}
				err2 := HandleAlertsQuery(&query)

				require.Nil(t, err2)
				require.Equal(t, 0, len(query.Result))
			})
		})
	})
}

func TestPausingAlerts(t *testing.T) {
	mockTimeNow()
	defer resetTimeNow()

	t.Run("Given an alert", func(t *testing.T) {
		sqlStore := InitTestDB(t)

		testDash := insertTestDashboard(t, sqlStore, "dashboard with alerts", 1, 0, false, "alert")
		alert, err := insertTestAlert("Alerting title", "Alerting message", testDash.OrgId, testDash.Id, simplejson.New())
		require.NoError(t, err)

		stateDateBeforePause := alert.NewStateDate
		stateDateAfterPause := stateDateBeforePause

		// Get alert so we can use its ID in tests
		alertQuery := models.GetAlertsQuery{DashboardIDs: []int64{testDash.Id}, PanelId: 1, OrgId: 1, User: &models.SignedInUser{OrgRole: models.ROLE_ADMIN}}
		err2 := HandleAlertsQuery(&alertQuery)
		require.Nil(t, err2)

		insertedAlert := alertQuery.Result[0]

		t.Run("when paused", func(t *testing.T) {
			_, err := pauseAlert(testDash.OrgId, insertedAlert.Id, true)
			require.NoError(t, err)

			t.Run("the NewStateDate should be updated", func(t *testing.T) {
				alert, err := getAlertById(insertedAlert.Id)
				require.NoError(t, err)

				stateDateAfterPause = alert.NewStateDate
				So(stateDateBeforePause, ShouldHappenBefore, stateDateAfterPause)
			})
		})

		t.Run("when unpaused", func(t *testing.T) {
			_, err := pauseAlert(testDash.OrgId, insertedAlert.Id, false)
			require.NoError(t, err)

			t.Run("the NewStateDate should be updated again", func(t *testing.T) {
				alert, err := getAlertById(insertedAlert.Id)
				require.NoError(t, err)

				stateDateAfterUnpause := alert.NewStateDate
				So(stateDateAfterPause, ShouldHappenBefore, stateDateAfterUnpause)
			})
		})
	})
}
func pauseAlert(orgId int64, alertId int64, pauseState bool) (int64, error) {
	cmd := &models.PauseAlertCommand{
		OrgId:    orgId,
		AlertIds: []int64{alertId},
		Paused:   pauseState,
	}
	err := PauseAlert(cmd)
	require.NoError(t, err)
	return cmd.ResultCount, err
}
func insertTestAlert(title string, message string, orgId int64, dashId int64, settings *simplejson.Json) (*models.Alert, error) {
	items := []*models.Alert{
		{
			PanelId:     1,
			DashboardId: dashId,
			OrgId:       orgId,
			Name:        title,
			Message:     message,
			Settings:    settings,
			Frequency:   1,
		},
	}

	cmd := models.SaveAlertsCommand{
		Alerts:      items,
		DashboardId: dashId,
		OrgId:       orgId,
		UserId:      1,
	}

	err := SaveAlerts(&cmd)
	return cmd.Alerts[0], err
}

func getAlertById(id int64) (*models.Alert, error) {
	q := &models.GetAlertByIdQuery{
		Id: id,
	}
	err := GetAlertById(q)
	require.NoError(t, err)
	return q.Result, err
}

func pauseAllAlerts(pauseState bool) error {
	cmd := &models.PauseAllAlertCommand{
		Paused: pauseState,
	}
	err := PauseAllAlerts(cmd)
	require.NoError(t, err)
	return err
}
