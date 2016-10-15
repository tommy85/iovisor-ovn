package ovnmonitor

import (
	"github.com/netgroup-polito/iovisor-ovn/config"
	"github.com/socketplane/libovsdb"
)

func MonitorOvsDb() (h *MonitorHandler) {

	//handler: one for each monitor instance
	handler := MonitorHandler{}

	//channel to notificate someone with new TableUpdates
	handler.Update = make(chan *libovsdb.TableUpdates)

	//channel buffered to notify the logic of new changes
	handler.BufupdateOvs = make(chan string, 10000)

	//Channel buffered to notify the logic fo new changes
	handler.MainLogicNotification = make(chan string, 100)

	//cache contan a map between string and libovsdb.Row
	cache := make(map[string]map[string]libovsdb.Row)
	handler.Cache = &cache

	ovsdb_sock := ""
	if config.Sandbox == true {
		// Sandbox Real Environment
		ovsdb_sock = config.OvsSock
		ovs, err := libovsdb.ConnectWithUnixSocket(config.OvsSock)
		handler.Db = ovs
		if err != nil {
			log.Errorf("unable to Connect to %s - %s\n", config.OvsSock, err)
			return
		}
	} else {
		//Openstack Real Environment
		ovsdb_sock = config.Ovs
		ovs, err := libovsdb.Connect(config.FromStringToIpPort(config.Ovs))
		handler.Db = ovs
		if err != nil {
			log.Errorf("unable to Connect to %s - %s\n", ovsdb_sock, err)
			return
		}
	}

	log.Noticef("starting ovs local monitor @ %s\n", ovsdb_sock)

	var notifier MyNotifier
	notifier.handler = &handler
	handler.Db.Register(notifier)

	var ovsDb_name = "Open_vSwitch"
	initial, err := handler.Db.MonitorAll(ovsDb_name, "")
	if err != nil {
		log.Errorf("unable to Monitor %s - %s\n", ovsDb_name, err)
		return
	}
	PopulateCache(&handler, *initial)

	go OvsLogicInit(&handler)
	go ovsMonitorFilter(&handler)
	//<-handler.Quit
	h = &handler
	return
}

func ovsMonitorFilter(h *MonitorHandler) {
	printTable := make(map[string]int)
	printTable["Interface"] = 1
	//printTable["Port"] = 1
	//printTable["Bridge"] = 1

	for {
		select {
		case currUpdate := <-h.Update:
			//PrintCache(h)

			//manage case of new update from db

			//for debug purposes, print the new rows added or modified
			//a copy of the whole db is in cache.

			for table, _ /*tableUpdate*/ := range currUpdate.Updates {
				if _, ok := printTable[table]; ok {
					//Notify ovslogic to update db structs!
					h.BufupdateOvs <- table

					// log.Noticef("update table: %s\n", table)
					// for uuid, row := range tableUpdate.Rows {
					// 	log.Noticef("UUID     : %s\n", uuid)
					//
					// 	newRow := row.New
					// 	PrintRow(newRow)
					// }
				}
			}
		}
	}
}
