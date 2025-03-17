# K8S SSA FTW (almost)

Server-side Apply is an evolution of
the "last applied configuration" and the "merge patch".

Happens on the server

Super easy-to-use on the client side

When merging the changes, it considers 4 things:

* the current state of the object
* the patch
* the schema of the object's type
* the ownership of individual fields in the object

```shell
watch -n1 "oc get cm --show-managed-fields -n ssa-test test-cm -o yaml"
```

## Create is Update

Create and update are handled the same similarly to how
`kubectl apply` works.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
  namespace: ssa-test
data:
  key: value

```

```go
cl.Patch(ctx, obj, client.Apply,
  client.FieldOwner("alice"))
```

## Ownership

The `metadata.managedFields` details the "ownership" of:

* fields
* map-keys
* entries in set-like arrays

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
  namespace: ssa-test
data:
  my-key: my-value

```

```go
cl.Patch(ctx, obj, client.Apply,
  client.FieldOwner("bob"))
```

## Ownership

If "someone" wants to update a field owned by
"someone else", the update fails.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
  namespace: ssa-test
data:
  key: my-value

```

```go
cl.Patch(ctx, obj, client.Apply,
  client.FieldOwner("bob"))
```

## Ownership

Ownership can be "shared" if two managers applied the same
value to the field in question.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
  namespace: ssa-test
data:
  key: value
  my-key: my-value

```

```go
cl.Patch(ctx, obj, client.Apply,
  client.FieldOwner("bob"))
```

## Ownership

Ownership can be forced. This will "reassign" the ownership
to the manager you want.

This is especially useful if you want to make sure that
the subset of fields that you care about really do have
the values you want, but you don't care about the rest of
the fields in the object.

(looking at you, service accounts, cluster IPs, etc.)

The client API in Go has nice support for it:

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
  namespace: ssa-test
data:
  key: my-value
  my-key: my-value
```

```go
cl.Patch(ctx, obj, client.Apply,
  client.FieldOwner("bob"),
  client.ForceOwnership)
```

## Ownership

If the manager only uses SSA, the deletion of no longer
managed fields is _also taken care of_.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
  namespace: ssa-test
data:
  key: my-value
```

```go
cl.Patch(ctx, obj, client.Apply,
  client.FieldOwner("bob"))
```

## Apply vs. Update

The `metadata.managedFields` records both SSA patches and
regular update/create calls.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
data:
  key: value

```

```go
cl.Delete(ctx, obj)
cl.Create(ctx, obj, client.FieldOwner("alice"))
```

## Apply vs. Update

Mixing the two approaches kinda works.

* Setting the fields conflicts if the previous value was done using `Update` even if the owner is the same (overridable by force).

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
data:
  key: "different value"

```

```go
cl.Patch(ctx, obj, client.Apply,
  client.FieldOwner("alice"))
```

## Apply vs. Update

Mixing the two approaches kinda works.

* A field managed by an `Update` operation IS NOT
  deleted when applying an SSA patch EVEN IF the owner is
  the same.
* Migrating the managed fields is required to move from "updates" to SSA.
  `Kubectl` does this automagically if it sees the `last-applied-configuration`.
* We will need to do a similar migration with our objects.

```yaml
apiVersion: v1
kind: ConfigMap
metadata:
  name: test-cm
data:
  another-key: value 

```

```go
cl.Patch(ctx, obj, client.Apply,
  client.FieldOwner("alice"))
```

## Why to use SSA in controllers

Largely avoids the need to retry the reconcile on
concurrent modification that the typical `GET-modify-PUT`
workflow suffers from.

## How to use SSA in controllers

The problem that needs to be solved is that the patch must
only contain the fields that the controller wants to manage
(and no other).

The controllers usually use the `GET-modify-PUT` workflow.

If you wanted to use `PATCH` instead of `PUT`, you'd need
to make sure that you:

* either ALWAYS set all the fields that the controller
  should set (otherwise you risk them being deleted on
  the resource by SSA),
* or you `Get` the resource and somehow extract the fields
  that already "belong to you".

For standard k8s types, there are `Extract*(obj, owner)`
functions in `k8s.io/client-go/applyconfigurations/...`
packages that implement the second bullet point above.

The support for this in CRDs is still not present in
`controller-gen` though, see [here](https://github.com/kubernetes-sigs/controller-tools/pull/818)

## How to use SSA in controllers

Don't (just yet).

## Why we want to use SSA for Kubesaw templates

Templates only contain the objects as we want them to look
like. We need to tolerate changes made to other fields.

This is exactly what SSA supports nicely.

It is not the `GET-modify-PUT` workflow of controllers.

## Where to learn more about SSA

* [K8s docs](https://kubernetes.io/docs/reference/using-api/server-side-apply/)
* [K8s blog](https://kubernetes.io/blog/2021/08/06/server-side-apply-ga/#using-server-side-apply-in-a-controller)
* The internet and LLMs (even though they're somewhat confused about the topic, as usual)
