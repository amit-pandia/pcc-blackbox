package main

import (
	"fmt"
	pcc "github.com/platinasystems/pcc-blackbox/lib"
	"github.com/platinasystems/tiles/pccserver/security/model"
	"testing"
	"time"
)

////
// TEST
////

// Generic test for more endpoints
func genericTest(
	t *testing.T,
	item string,
	addItem func(name string, description string) (*pcc.GenericModel, error),
	getItem func(id uint64) (*pcc.GenericModel, error),
	deleteItem func(id uint64) error,
	listItems func() ([]pcc.GenericModel, error)) {
	var (
		items []pcc.GenericModel
		added *pcc.GenericModel
		get   *pcc.GenericModel
		err   error
	)

	if addItem != nil {
		name := fmt.Sprintf("%s-%d", "bb_test", time.Now().Unix())
		fmt.Println(fmt.Sprintf("adding the [%s] [%s]", item, name))
		if added, err = addItem(name, "blackbox test"); err == nil { // add the item
			fmt.Println(fmt.Sprintf("added the [%s] [%v]", item, *added))

			if getItem != nil { // get an item and check if it is the same with added
				if get, err = getItem(added.GetId()); err == nil {
					if get.GetDescription() == added.GetDescription() && get.GetName() == added.GetName() {
						fmt.Println(fmt.Sprintf("the fetched %s [%s] is ok", item, name))
					} else {
						t.Fatal(fmt.Errorf("the %s %d is different than expected", item, added.GetId()))
					}
				} else {
					t.Fatal(fmt.Sprintf("error getting the %s with id %d", item, added.GetId()), err)
				}
			}

			if deleteItem != nil { // delete the added item
				if err = deleteItem(added.GetId()); err == nil {
					fmt.Println(fmt.Sprintf("deleted the %s %d %s", item, added.GetId(), added.GetName()))
				} else {
					t.Fatal(fmt.Sprintf("error deleting the %s with id %d", item, added.GetId()), err)
				}
			}
		} else {
			t.Fatal(err)
		}
	}

	// get all items
	if listItems != nil {
		if items, err = listItems(); err == nil {
			fmt.Println(fmt.Sprintf("found %d %s", len(items), item))
		} else {
			t.Fatal(err)
		}
	}
}

// add, get and delete an entity. get all entities
func testUMEntity(t *testing.T) {
	genericTest(t,
		"entity",
		func(name string, description string) (genericModel *pcc.GenericModel, err error) {
			return pcc.Pcc.AddEntity(name, description)
		},
		func(id uint64) (genericModel *pcc.GenericModel, err error) {
			return pcc.Pcc.GetEntity(id)
		},
		nil,
		func() (models []pcc.GenericModel, err error) {
			return pcc.Pcc.GetEntities()
		},
	)
}

// add, get and delete a role. get all roles
// test the deletion of a role still used
func testUMRole(t *testing.T) {
	genericTest(t,
		"role",
		func(name string, description string) (genericModel *pcc.GenericModel, err error) {
			return pcc.Pcc.AddRole(name, description)
		},
		func(id uint64) (genericModel *pcc.GenericModel, err error) {
			return pcc.Pcc.GetSecurityRole(id)
		},
		func(id uint64) error {
			return pcc.Pcc.DeleteRole(id)
		}, func() (models []pcc.GenericModel, err error) {
			return pcc.Pcc.GetSecurityRoles()
		},
	)

	// test the deletion of a role still used
	if user, err := addTestUser(); err == nil {
		defer pcc.Pcc.DelUser(user.UserName)
		if errOk := pcc.Pcc.DeleteRole(user.Role.Id); errOk == nil {
			t.Fatal(fmt.Sprintf("the role [%d] has been deleted", user.Role.Id))
		} else {
			fmt.Println(fmt.Sprintf("the system does not allowed to delete the role [%d]. [%s]", user.Role.Id, errOk.Error()))
		}
	} else {
		t.Fatal(err)
	}
}

func testUMOperation(t *testing.T) {
	// Convert operation to Generic Model
	convert := func(f func() ([]pcc.Operation, error)) (models []pcc.GenericModel, err error) {
		if operations, e := f(); e == nil {
			for _, operation := range operations {
				models = append(models, operation.GenericModel)
			}
		} else {
			err = e
		}
		return
	}

	genericTest(t,
		"operation",
		nil, nil, nil,
		func() (models []pcc.GenericModel, err error) {
			return convert(pcc.Pcc.GetOperations)
		},
	)

	genericTest(t,
		"groupoperation",
		nil, nil, nil,
		func() (models []pcc.GenericModel, err error) {
			return convert(pcc.Pcc.GetGroupOperations)
		},
	)
}

// add a test user
func addTestUser() (user *pcc.User, err error) {
	name := fmt.Sprintf("%d@platinasystems.net", time.Now().Unix())
	u := pcc.User{UserName: name, FirstName: "John", LastName: "Wick"}

	var roles []pcc.GenericModel

	if roles, err = pcc.Pcc.GetSecurityRoles(); err == nil {
		u.RoleId = roles[0].Id
	} else {
		return
	}

	if user, err = pcc.Pcc.AddUser(u); err == nil {
		fmt.Println(fmt.Sprintf("added the user [%d:%s] with role [%+v] and tenant [%+v]", user.Id, user.UserName, user.Role, user.Tenant))
	}
	return
}

// add and delete an user.
// add an existing user and check for the error.
// get all users.
func testUMUser(t *testing.T) {
	if added, err := addTestUser(); err == nil {
		user2 := *added
		fmt.Println(fmt.Sprintf("trying add the same user %s", user2.UserName))
		if _, errOk := pcc.Pcc.AddUser(user2); errOk == nil {
			t.Fatal("the duplicate user was added without issues")
		} else {
			fmt.Println(fmt.Sprintf("the system does not allowed to re-add the user [%s]. [%s]", added.UserName, errOk.Error()))
		}

		if err = pcc.Pcc.DelUser(added.UserName); err == nil {
			fmt.Println(fmt.Sprintf("deleted the user %d %s", added.Id, added.UserName))
		} else {
			t.Fatal(err)
		}
	} else {
		t.Fatal(err)
	}

	if users, err := pcc.Pcc.GetUsers(); err == nil {
		fmt.Println(fmt.Sprintf("found %d users", len(users)))
	} else {
		t.Fatal(err)
	}

}

// add, get and delete a tenant. get all tenants
// try to delete a tenant still used and check for the error
func testUMTenant(t *testing.T) {
	var err error

	name := fmt.Sprintf("%s-%d", "bb_test", time.Now().Unix())
	tenant := &(pcc.Tenant{Tenant: model.Tenant{Name: name, Description: "blackbox test"}})
	fmt.Println(fmt.Sprintf("adding the tenant %s", name))
	if tenant, err = pcc.Pcc.AddTenant(*tenant); err == nil {
		fmt.Println(fmt.Sprintf("added the tenant %v", *tenant))

		if tenant, err = pcc.Pcc.GetTenant(tenant.ID); err == nil {
			if tenant.Name == name {
				fmt.Println(fmt.Sprintf("the fetched tenant %s is ok", name))
			} else {
				t.Fatal(fmt.Sprintf("the tenant %d is different than expected", tenant.ID))
			}
		} else {
			t.Fatal(err)
		}

		if err = pcc.Pcc.DelTenant(tenant.ID); err == nil {
			fmt.Println(fmt.Sprintf("deleted the tenant %d %s", tenant.ID, tenant.Name))
		} else {
			t.Fatal(err)
		}
	} else {
		t.Fatal(err)
	}

	// get all tenants
	if tenants, err := pcc.Pcc.GetTenants(); err == nil {
		fmt.Println(fmt.Sprintf("found %d tenants", len(tenants)))
	} else {
		t.Fatal(err)
	}

	// test the deletion of a tenant still used
	if user, err := addTestUser(); err == nil {
		defer pcc.Pcc.DelUser(user.UserName)
		if errOk := pcc.Pcc.DelTenant(user.Tenant.ID); errOk == nil {
			t.Fatal(fmt.Sprintf("the tenant has been deleted [%d]", user.TenantId))
		} else {
			fmt.Println(fmt.Sprintf("the system does not allowed to delete the tenant [%d]. [%s]", user.TenantId, errOk.Error()))
		}
	} else {
		t.Fatal(err)
	}
}

// add, get and delete an user space.
func testUMUserSpace(t *testing.T) {
	scope := "bb-test"
	content := "blackbox test"
	if err := pcc.Pcc.SetUserSpace("bb-test", content); err == nil {
		fmt.Println(fmt.Sprintf("added the user space [%s]", scope))

		var get string
		if get, err = pcc.Pcc.GetUserSpace(scope); err == nil {
			if get == content {
				fmt.Println(fmt.Sprintf("the fetched user space %s is ok", scope))
			} else {
				t.Fatal(fmt.Sprintf("the user space %s is different than expected %s", scope, get))
			}
		} else {
			t.Fatal(err)
		}
		if err = pcc.Pcc.DeleteUserSpace(scope); err == nil {
			fmt.Println(fmt.Sprintf("deleted the user space %s", scope))
		} else {
			t.Fatal(err)
		}
	} else {
		t.Fatal(err)
	}

}
