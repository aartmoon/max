package db

import (
	"context"
	"fmt"
)

func (s *Store) SeedDemo(ctx context.Context) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin GAR demo seed: %w", err)
	}
	defer tx.Rollback(ctx)
	statements := []string{
		`INSERT INTO gar_address_objects(id,object_id,object_guid,change_id,name,type_name,level,update_date,start_date,end_date,is_actual,is_active,raw_attributes) VALUES
          (1,1000,'10000000-0000-0000-0000-000000001000',1,'Москва','г',1,current_date,current_date,'2079-06-06',true,true,'{}'),
          (2,1001,'10000000-0000-0000-0000-000000001001',1,'Тверская','ул',8,current_date,current_date,'2079-06-06',true,true,'{}'),
          (3,1002,'10000000-0000-0000-0000-000000001002',1,'Арбат','ул',8,current_date,current_date,'2079-06-06',true,true,'{}')`,
		`INSERT INTO gar_houses(id,object_id,object_guid,change_id,house_num,update_date,start_date,end_date,is_actual,is_active,raw_attributes) VALUES
          (1,2000,'20000000-0000-0000-0000-000000002000',1,'1',current_date,current_date,'2079-06-06',true,true,'{}'),
          (2,2001,'20000000-0000-0000-0000-000000002001',1,'12',current_date,current_date,'2079-06-06',true,true,'{}')`,
		`INSERT INTO gar_apartments(id,object_id,object_guid,change_id,number,apart_number,update_date,start_date,end_date,is_actual,is_active,raw_attributes) VALUES
          (1,3000,'30000000-0000-0000-0000-000000003000',1,'1','1',current_date,current_date,'2079-06-06',true,true,'{}'),
          (2,3001,'30000000-0000-0000-0000-000000003001',1,'2','2',current_date,current_date,'2079-06-06',true,true,'{}'),
          (3,3002,'30000000-0000-0000-0000-000000003002',1,'15','15',current_date,current_date,'2079-06-06',true,true,'{}')`,
		`INSERT INTO gar_adm_hierarchy(id,object_id,parent_object_id,change_id,path,update_date,start_date,end_date,is_active,raw_attributes) VALUES
          (1,1000,NULL,1,'1000',current_date,current_date,'2079-06-06',true,'{}'),
          (2,1001,1000,1,'1000.1001',current_date,current_date,'2079-06-06',true,'{}'),
          (3,1002,1000,1,'1000.1002',current_date,current_date,'2079-06-06',true,'{}'),
          (4,2000,1001,1,'1000.1001.2000',current_date,current_date,'2079-06-06',true,'{}'),
          (5,2001,1002,1,'1000.1002.2001',current_date,current_date,'2079-06-06',true,'{}'),
          (6,3000,2000,1,'1000.1001.2000.3000',current_date,current_date,'2079-06-06',true,'{}'),
          (7,3001,2000,1,'1000.1001.2000.3001',current_date,current_date,'2079-06-06',true,'{}'),
          (8,3002,2001,1,'1000.1002.2001.3002',current_date,current_date,'2079-06-06',true,'{}')`,
	}
	for _, statement := range statements {
		if _, err := tx.Exec(ctx, statement); err != nil {
			return fmt.Errorf("seed GAR demo data: %w", err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit GAR demo seed: %w", err)
	}
	return nil
}
