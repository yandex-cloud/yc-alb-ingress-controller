package k8s

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

func TestLoader_Basic(t *testing.T) {
	active := ingWithName("active")

	deleted := ingWithName("deleted")
	deleted.Finalizers = append(deleted.Finalizers, Finalizer)
	deleted.DeletionTimestamp = &metav1.Time{Time: time.Now()}

	otherGroup := ingWithName("other-group")
	otherGroup.Annotations = map[string]string{
		AlbTag: "non-default",
	}

	objects := []client.Object{&active, &deleted, &otherGroup}
	exp := IngressGroup{
		Tag:     "default",
		Items:   []v1.Ingress{active},
		Deleted: []v1.Ingress{deleted},
	}

	cli := fake.NewClientBuilder().WithObjects(objects...).Build()
	loader := NewGroupLoader(cli)

	act, err := loader.Load(context.Background(), types.NamespacedName{Name: "default"})
	assert.NoError(t, err)

	assertGroupsEqual(t, exp, *act)
}

func TestLoader_WithOrder(t *testing.T) {
	empty := v1.Ingress{}
	empty.ObjectMeta.Annotations = map[string]string{AlbTag: "default"}
	ing10 := ingWithOrder("1")
	ing11 := ingWithOrder("1")
	ing2 := ingWithOrder("2")

	ing10.Name = "first"
	ing11.Name = "second"

	ingToBig := ingWithOrder("1000000")
	ingNeg := ingWithOrder("-45")
	ingGarbage := ingWithOrder("garbage")

	testData := []struct {
		desc    string
		ings    []client.Object
		exp     []v1.Ingress
		wantErr bool
	}{
		{
			desc: "basic",
			ings: []client.Object{&ing2, &ing10, &empty, &ing11},
			exp:  []v1.Ingress{empty, ing10, ing11, ing2},
		},
		{
			desc: "negative order",
			ings: []client.Object{&ing2, &ing10, &empty, &ing11, &ingNeg},
			exp:  []v1.Ingress{ingNeg, empty, ing10, ing11, ing2},
		},
		{
			desc:    "to big order",
			ings:    []client.Object{&ing2, &ing10, &empty, &ing11, &ingToBig},
			wantErr: true,
		},
		{
			desc:    "garbage order",
			ings:    []client.Object{&ing2, &ing10, &empty, &ing11, &ingGarbage},
			wantErr: true,
		},
	}

	for _, entry := range testData {
		t.Run(entry.desc, func(t *testing.T) {
			i := entry.ings

			cli := fake.NewClientBuilder().WithObjects(i...).Build()
			loader := NewGroupLoader(cli)

			g, err := loader.Load(context.Background(), types.NamespacedName{Name: "default"})
			if entry.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assertGroupsEqual(t, IngressGroup{Items: entry.exp}, *g)
		})
	}
}

func TestLoader_LoadWithClasses(t *testing.T) {
	defaultClass := v1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default",
			Annotations: map[string]string{
				DefaultIngressClass: "true",
			},
		},
		Spec: v1.IngressClassSpec{
			Controller: ControllerName,
		},
	}

	notdefaultClass := v1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "not-default",
		},
		Spec: v1.IngressClassSpec{
			Controller: ControllerName,
		},
	}

	defaultClassWrongControllerName := v1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "default-wrong",
			Annotations: map[string]string{
				DefaultIngressClass: "true",
			},
		},
		Spec: v1.IngressClassSpec{
			Controller: "klubnika",
		},
	}

	notdefaultClassWrongControllerName := v1.IngressClass{
		ObjectMeta: metav1.ObjectMeta{
			Name: "not-default-wrong",
		},
		Spec: v1.IngressClassSpec{
			Controller: "ananas",
		},
	}

	ingWithoutClass := ingWithName("without-class")
	correctIng := ingWithClass("not-default")
	correctIngDefault := ingWithClass("default")
	incorrectIng := ingWithClass("not-default-wrong")
	noMoreManagedIng := ingWithClass("not-default-wrong")
	noMoreManagedIng.Name = "no-more-managed"
	noMoreManagedIng.Finalizers = append(noMoreManagedIng.Finalizers, Finalizer)

	noMoreManagedIngNotDefault := ingWithClass("not-default-wrong")
	noMoreManagedIngNotDefault.ObjectMeta.Annotations[AlbTag] = "not-default"
	noMoreManagedIngNotDefault.Finalizers = append(noMoreManagedIngNotDefault.Finalizers, Finalizer)

	ingWithoutAlbTag := ingWithName("without-alb-tag")
	ingWithoutAlbTag.Finalizers = append(ingWithoutAlbTag.Finalizers, Finalizer)
	delete(ingWithoutAlbTag.ObjectMeta.Annotations, AlbTag)

	testData := []struct {
		desc    string
		objs    []client.Object
		exp     IngressGroup
		wantErr bool
	}{
		{
			desc: "without-class",
			objs: []client.Object{&ingWithoutClass},
			exp:  IngressGroup{Items: []v1.Ingress{ingWithoutClass}},
		},
		{
			desc: "with-correct-default-class",
			objs: []client.Object{&ingWithoutClass, &defaultClass},
			exp:  IngressGroup{Items: []v1.Ingress{ingWithoutClass}},
		},
		{
			desc: "with-incorrect-default-class",
			objs: []client.Object{&ingWithoutClass, &defaultClassWrongControllerName},
			exp:  IngressGroup{Items: []v1.Ingress{}},
		},
		{
			desc: "with-correct-class",
			objs: []client.Object{&ingWithoutClass, &correctIng, &incorrectIng, &notdefaultClass, &notdefaultClassWrongControllerName},
			exp:  IngressGroup{Items: []v1.Ingress{correctIng, ingWithoutClass}},
		},
		{
			desc: "with-other-class",
			objs: []client.Object{&ingWithoutClass, &notdefaultClassWrongControllerName},
			exp:  IngressGroup{Items: []v1.Ingress{ingWithoutClass}},
		},
		{
			desc: "with-no-more-managed-ingress",
			objs: []client.Object{&noMoreManagedIng, &notdefaultClassWrongControllerName},
			exp:  IngressGroup{Deleted: []v1.Ingress{noMoreManagedIng}},
		},
		{
			desc: "mixed",
			objs: []client.Object{
				&ingWithoutClass, &correctIng, &incorrectIng, &correctIngDefault, &noMoreManagedIng,
				&notdefaultClass, &notdefaultClassWrongControllerName, &defaultClass, &ingWithoutAlbTag,
			},
			exp: IngressGroup{
				Items:   []v1.Ingress{correctIngDefault, correctIng, ingWithoutClass},
				Deleted: []v1.Ingress{noMoreManagedIng, ingWithoutAlbTag},
			},
		},
		{
			desc: "with-no-more-managed-not-default-ingress",
			objs: []client.Object{&noMoreManagedIng, &noMoreManagedIngNotDefault},
			exp:  IngressGroup{Deleted: []v1.Ingress{noMoreManagedIng}},
		},
		{
			desc: "without-alb-tag",
			objs: []client.Object{&ingWithoutAlbTag, &notdefaultClass},
			exp:  IngressGroup{Deleted: []v1.Ingress{ingWithoutAlbTag}},
		},
	}
	for _, entry := range testData {
		t.Run(entry.desc, func(t *testing.T) {
			cli := fake.NewClientBuilder().WithObjects(entry.objs...).Build()
			loader := NewGroupLoader(cli)

			g, err := loader.Load(context.Background(), types.NamespacedName{Name: "default"})
			if entry.wantErr {
				assert.Error(t, err)
				return
			}

			assert.NoError(t, err)
			assertGroupsEqual(t, entry.exp, *g)
		})
	}
}

func ingWithClass(class string) v1.Ingress {
	return v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ingress-with-class-" + class,
			Namespace: "default",
			Annotations: map[string]string{
				AlbTag: "default",
			},
		},
		Spec: v1.IngressSpec{
			IngressClassName: &class,
		},
	}
}

func ingWithName(name string) v1.Ingress {
	return v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			Annotations: map[string]string{
				AlbTag: "default",
			},
		},
	}
}

func ingWithOrder(order string) v1.Ingress {
	return v1.Ingress{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ingress-with-order-" + order,
			Namespace: "default",
			Annotations: map[string]string{
				OrderInGroup: order,
				AlbTag:       "default",
			},
		},
	}
}

func assertGroupsEqual(t *testing.T, exp, act IngressGroup) {
	t.Helper()

	assert.Equal(t, len(exp.Items), len(act.Items))
	for i := range act.Items {
		assert.Equal(t, exp.Items[i].Name, act.Items[i].Name)
	}

	assert.Equal(t, len(exp.Deleted), len(act.Deleted))
	for i := range act.Deleted {
		assert.Equal(t, exp.Deleted[i].Name, act.Deleted[i].Name)
	}
}
