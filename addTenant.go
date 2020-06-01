package main

import (
	"fmt"
	pcc "github.com/platinasystems/pcc-blackbox/lib"
	"github.com/platinasystems/test"
	"testing"
)

func addTenant(t *testing.T) {
	t.Run("addTenant", addTenantA)
}

func addTenantA(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}
	var (
		tenants []pcc.Tenant
		tenant  pcc.Tenant
		tenant2 pcc.Tenant
		addReq  pcc.Tenant
		addReq2 pcc.Tenant
		err     error
	)

	fmt.Println("assign all nodes to ROOT")
	var nodes []uint64
	for _, i := range Env.Servers {
		nodes = append(nodes, NodebyHostIP[i.HostIp])
	}
	err = pcc.Pcc.AssignTenantNodes(1, nodes)
	if err != nil {
		assert.Fatalf("%v\n", err)
	}

	fmt.Println("Delete existing tenants")
	tenants, err = pcc.Pcc.GetTenants()
	if err != nil {
		assert.Fatalf("%v\n", err)
	}
	for _, t := range tenants {
		fmt.Printf("delete tenant %v\n", t.Name)
		pcc.Pcc.DelTenant(t.ID)
	}

	addReq = pcc.Tenant{}
	addReq.Name = "cust-a"
	addReq.Description = "a tenant of ROOT"

	fmt.Printf("add tenant %v\n", addReq.Name)
	_, err = pcc.Pcc.AddTenant(addReq)
	if err != nil {
		assert.Fatalf("Failed to add tenant %v: %v\n",
			addReq.Name, err)
	}

	fmt.Printf("find tenant %v\n", addReq.Name)
	tenant, err = pcc.Pcc.FindTenant(addReq.Name)
	if err != nil {
		assert.Fatalf("%v\n", err)
	}
	fmt.Printf("tenant %v, id %v\n", tenant.Name, tenant.ID)

	addReq2.Name = "cust-b"
	addReq2.Description = "a tenant of cust-b"
	addReq2.Parent = tenant.ID

	fmt.Printf("add tenant %v\n", addReq.Name)
	_, err = pcc.Pcc.AddTenant(addReq2)
	if err != nil {
		assert.Fatalf("%v\n", err)
	}

	fmt.Printf("find tenant %v\n", addReq2.Name)
	tenant2, err = pcc.Pcc.FindTenant(addReq2.Name)
	if err != nil {
		assert.Fatalf("%v\n", err)
	}
	fmt.Printf("tenant %v, id %v\n", tenant2.Name, tenant2.ID)

	fmt.Printf("deleting tenant %v\n", tenant2.Name)
	err = pcc.Pcc.DelTenant(tenant2.ID)
	if err != nil {
		assert.Fatalf("%v\n", err)
	} else {
	}
	_, err = pcc.Pcc.FindTenant(addReq2.Name)
	if err != nil {
		fmt.Printf("FindTenant failed as exepted on deleted tenant\n")
	} else {
		assert.Fatalf("%v\n", "Expecting failure, but didn't")
	}

	fmt.Printf("assign servers to tenant %v\n", addReq.Name)
	for _, i := range Env.Servers {
		nodes = append(nodes, NodebyHostIP[i.HostIp])
	}
	err = pcc.Pcc.AssignTenantNodes(tenant.ID, nodes)
	if err != nil {
		assert.Fatalf("%v\n", err)
	}

	source := fmt.Sprintf("https://%v:7654/setPass", Env.PccIp)

	// remove existing users
	if users, err := pcc.Pcc.GetUsers(); err != nil {
		assert.Fatalf("Failed to get users: %v\n", err)
		return
	} else {
		for _, u := range users {
			if u.UserName == "admin" {
				continue
			}
			err = pcc.Pcc.DelUser(u.UserName)
			if err != nil {
				assert.Fatalf("Failed to delete user %v: %v\n",
					u.UserName, err)
				return
			}
		}
	}

	addUser := pcc.User{
		UserName:  "bsimpson@platinasystems.com",
		FirstName: "Bart",
		LastName:  "Simpson",
		Email:     "bsimpson@platinasystems.com",
		Password:  "lisasux",
		Active:    true,
		Protect:   false,
		RoleId:    1,
		TenantId:  tenant.ID,
		Source:    source,
	}
	_, err = pcc.Pcc.AddUser(addUser)
	if err != nil {
		assert.Fatalf("Failed to add user %v: %v\n", addUser.UserName,
			err)
		return
	}

	addUser2 := pcc.User{
		UserName:  "lsimpson@platinasystems.com",
		FirstName: "Lisa",
		LastName:  "Simpson",
		Email:     "lsimpson@platinasystems.com",
		Password:  "bartsux",
		Active:    true,
		Protect:   false,
		RoleId:    1,
		TenantId:  tenant.ID,
		Source:    source,
	}
	if _, err = pcc.Pcc.AddUser(addUser2); err != nil {
		assert.Fatalf("Failed to add user %v: %v\n", addUser2.UserName,
			err)
		return
	}

	fmt.Printf("Try change firstname of user %v \n", addUser.UserName)
	newName := "Mr Bart"
	addUser.LastName = newName

	if err = pcc.Pcc.UpdateUser(addUser); err != nil {
		assert.Fatalf("Failed to update user %v: %v\n", newName, err)
		return
	}

	if users, err := pcc.Pcc.GetUsers(); err == nil {
		found := false
		for _, u := range users {
			if u.UserName == addUser.UserName {
				fmt.Printf("Found updated user %v\n", u)
				if u.Profile.LastName == newName {
					fmt.Printf("user update worked\n")
					found = true
				} else {
					assert.Fatalf("user update failed\n")
					return
				}
			}
		}
		if !found {
			assert.Fatalf("user update failed and not found\n")
			return
		}
	} else {
		assert.Fatalf("%v\n", err)
	}

	err = pcc.Pcc.DelUser(addUser.UserName)
	if err != nil {
		assert.Fatalf("%v\n", err)
	}
}

func delAllTenants(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}
	var (
		tenants []pcc.Tenant
		err     error
	)

	if tenants, err = pcc.Pcc.GetTenants(); err == nil {
		for _, t := range tenants {
			if t.Protect {
				continue
			}
			id := t.ID
			if err = pcc.Pcc.DelTenant(id); err != nil {
				assert.Fatalf("Failed to DelTenant %v: %v\n", id, err)
				return
			}
		}
	} else {
		assert.Fatalf("Failed to GetTenants: %v\n", err)
	}
}

func delAllUsers(t *testing.T) {
	test.SkipIfDryRun(t)
	assert := test.Assert{t}
	var (
		users []pcc.User
		err   error
	)

	if users, err = pcc.Pcc.GetUsers(); err == nil {
		for _, u := range users {
			if u.Protect {
				continue
			}
			username := u.UserName
			if err = pcc.Pcc.DelUser(username); err != nil {
				assert.Fatalf("failed to Delete user %v: %v\n", username, err)
				return
			}
		}
	} else {
		assert.Fatalf("Failed to GetUsers: %v\n", err)
		return
	}
}
