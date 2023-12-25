package podcast

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/sschwartz96/syncapod-backend/internal/db_new"
)

type CategoryCache struct {
	dbCats  []db_new.Category // index represents id
	codes   map[string]int32  // codes[key] = parentID; where key is generate by buildAncesterTree()
	mutex   sync.RWMutex      // allows CategoryCache to be thread safe
	queries *db_new.Queries
}

func newCategoryCache(dbCats []db_new.Category, queries *db_new.Queries) *CategoryCache {
	catCache := CategoryCache{
		dbCats:  make([]db_new.Category, 0),
		codes:   make(map[string]int32),
		mutex:   sync.RWMutex{},
		queries: queries,
	}
	catCache.dbCats = append(catCache.dbCats, dbCats...)
	for i := range dbCats {
		catCache.codes[catCache.buildAncesterTree(dbCats[i].ParentID, dbCats[i].Name)] = dbCats[i].ID
	}
	return &catCache
}

// LookupIDs takes array of category ids, returns an array of Category
// with their respective sub-categories, max recursive depth of Category is 2
// parent categories MUST come before their children
func (c *CategoryCache) LookupIDs(ids []int32) ([]Category, error) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	parentMap := map[int32]*Category{}
	// range through all ids
	for i := range ids {
		if i > len(c.dbCats) {
			return nil, errors.New("CategoryCache.LookupIDs() error: category index out of range")
		}
		dbCat := c.dbCats[ids[i]]
		// no parent means it is a parent, create new parent cat
		if dbCat.ParentID == 0 {
			parentMap[dbCat.ID] = &Category{ID: dbCat.ID, Name: dbCat.Name, Subcategories: []Category{}}
			continue
		}
		// check to make sure we have a valid sub category
		parent, ok := parentMap[dbCat.ParentID]
		if !ok {
			return nil, fmt.Errorf("CategoryCache.LookupIDs() error: parent map does not exist,catID: %d, parentID: %d", dbCat.ID, dbCat.ParentID)
		}
		// append to existing parent
		parent.Subcategories = append(parent.Subcategories, Category{dbCat.ID, dbCat.Name, nil})
	}
	// aggregate parents into category slice
	cats := []Category{}
	for _, c := range parentMap {
		cats = append(cats, *c)
	}
	return catSort(cats), nil
}

// TranslateCategories recursively appends category ids into a slice of ids
// Uses the codes maps held within the CategoryCache
func (c *CategoryCache) TranslateCategories(cats []Category) ([]int32, error) {
	var catIDs []int32
	var unknown string
	// translateCategories but on category does not exist restart
	for catIDs, unknown = c.translateCategories(cats, 0, []int32{}); unknown != ""; catIDs, unknown = c.translateCategories(cats, 0, []int32{}) {
		if err := c.addNewCategory(unknown, catIDs[0]); err != nil {
			return nil, err
		}
	}
	return catIDs, nil
}

// translateCategories return succesful translated categories or name of unknown category
func (c *CategoryCache) translateCategories(cats []Category, parentID int32, ids []int32) ([]int32, string) {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if cats == nil {
		return ids, ""
	}
	for i := range cats {
		var tree string
		cat := cats[i]
		// if current category is top of hierarchy
		if parentID == 0 {
			tree = cat.Name
		} else {
			tree = c.buildAncesterTree(parentID, cat.Name)
			// the current category does not exist in database
			if c.codes[tree] == 0 {
				return []int32{parentID}, cat.Name
			}
		}

		// append parent id
		ids = append(ids, c.codes[tree])

		// recursively append children
		unk := ""
		ids, unk = c.translateCategories(cat.Subcategories, c.codes[tree], ids)
		if unk != "" {
			return ids, unk
		}
	}
	return ids, ""
}

// addNewCategory takes a category name and parent id to construct new category
// saves new category into database and return current id
// returns error if database connection fails
func (c *CategoryCache) addNewCategory(name string, parentID int32) error {
	// construct new category
	cat := db_new.Category{ID: int32(len(c.dbCats)), Name: name, ParentID: int32(parentID)}
	tree := c.buildAncesterTree(parentID, cat.Name)

	// lock cache and update
	c.mutex.Lock()
	c.dbCats = append(c.dbCats, cat)
	c.codes[tree] = cat.ID
	c.mutex.Unlock()

	// insert into db
	ctx, cncFn := context.WithTimeout(context.Background(), time.Second*5)
	defer cncFn()
	err := c.queries.InsertCategory(ctx, db_new.InsertCategoryParams(cat))
	if err != nil {
		return fmt.Errorf("addNewCategory() error: %v", err)
	}
	return nil
}

func catSort(c []Category) []Category {
	sort.Slice(c, func(i, j int) bool {
		return c[i].ID < c[j].ID
	})
	return c
}

// buildAncesterTree takes:
//
//	pid: parent id
//	s: cat name
//
// returns string in form of: etc->grandparent->parent->child(current)
func (c *CategoryCache) buildAncesterTree(pid int32, s string) string {
	c.mutex.RLock()
	defer c.mutex.RUnlock()
	if pid == 0 {
		return s
	}
	return c.buildAncesterTree(c.dbCats[pid].ParentID, c.dbCats[pid].Name) + s
}
